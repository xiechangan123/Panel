package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3AddressingStyle S3 地址模式
type S3AddressingStyle string

const (
	// S3AddressingStylePath Path 模式：https://s3.region.amazonaws.com/bucket/key
	S3AddressingStylePath S3AddressingStyle = "path"
	// S3AddressingStyleVirtualHosted Virtual Hosted 模式：https://bucket.s3.region.amazonaws.com/key
	S3AddressingStyleVirtualHosted S3AddressingStyle = "virtual-hosted"
)

type S3Config struct {
	Region          string            // AWS 区域
	Bucket          string            // S3 存储桶名称
	AccessKeyID     string            // 访问密钥 ID
	SecretAccessKey string            // 访问密钥
	Endpoint        string            // 自定义端点
	BasePath        string            // 基础路径前缀
	AddressingStyle S3AddressingStyle // 地址模式
}

type S3 struct {
	client *s3.Client
	config S3Config
}

func NewS3(cfg S3Config) (Storage, error) {
	if cfg.AddressingStyle == "" {
		cfg.AddressingStyle = S3AddressingStyleVirtualHosted
	}

	cfg.BasePath = strings.Trim(cfg.BasePath, "/")

	var awsCfg aws.Config
	var err error

	awsCfg, err = config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
		config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
		config.WithRetryMaxAttempts(10),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *s3.Client
	if cfg.Endpoint != "" {
		// 自定义端点
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = cfg.AddressingStyle == S3AddressingStylePath
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	} else {
		// 标准 AWS S3
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = cfg.AddressingStyle == S3AddressingStylePath
		})
	}

	return &S3{
		client: client,
		config: cfg,
	}, nil
}

// Delete 删除文件
func (s *S3) Delete(files ...string) error {
	if len(files) == 0 {
		return nil
	}

	// 批量删除
	var objects []types.ObjectIdentifier
	for _, file := range files {
		key := s.getKey(file)
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	_, err := s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(s.config.Bucket),
		Delete: &types.Delete{
			Objects: objects,
		},
	})

	waiter := s3.NewObjectNotExistsWaiter(s.client)
	for _, file := range files {
		key := s.getKey(file)
		err = waiter.Wait(context.TODO(), &s3.HeadObjectInput{
			Bucket: aws.String(s.config.Bucket),
			Key:    aws.String(key),
		}, 30*time.Second)
		if err != nil {
			return err
		}
	}

	return err
}

// Exists 检查文件是否存在
func (s *S3) Exists(file string) bool {
	key := s.getKey(file)
	_, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

// LastModified 获取文件最后修改时间
func (s *S3) LastModified(file string) (time.Time, error) {
	key := s.getKey(file)
	output, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return time.Time{}, err
	}

	if output.LastModified != nil {
		return *output.LastModified, nil
	}
	return time.Time{}, nil
}

// List 列出目录下的所有文件
func (s *S3) List(path string) ([]string, error) {
	prefix := s.getKey(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var files []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.config.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			// 跳过目录本身
			if key == prefix {
				continue
			}
			// 提取文件名
			name := strings.TrimPrefix(key, prefix)
			if name != "" && !strings.Contains(name, "/") {
				files = append(files, name)
			}
		}
	}

	return files, nil
}

// Put 写入文件内容
func (s *S3) Put(file string, content io.Reader) error {
	key := s.getKey(file)

	// For S3-compatible providers, disable automatic checksum calculation on the Uploader.
	// The S3 client's RequestChecksumCalculation setting only affects single-part uploads.
	// Multipart uploads via the Uploader require this separate setting (added in s3/manager v1.20.0).
	// See: https://github.com/aws/aws-sdk-go-v2/issues/3007
	uploader := manager.NewUploader(s.client, func(u *manager.Uploader) {
		u.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   content,
	})
	if err != nil {
		return err
	}

	waiter := s3.NewObjectExistsWaiter(s.client)
	err = waiter.Wait(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	}, 30*time.Second)

	return err
}

// Size 获取文件大小
func (s *S3) Size(file string) (int64, error) {
	key := s.getKey(file)
	output, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, err
	}

	return aws.ToInt64(output.ContentLength), nil
}

// getKey 获取完整的对象键
func (s *S3) getKey(file string) string {
	file = strings.TrimPrefix(file, "/")
	if s.config.BasePath == "" {
		return file
	}
	if file == "" {
		return s.config.BasePath
	}
	return fmt.Sprintf("%s/%s", s.config.BasePath, file)
}
