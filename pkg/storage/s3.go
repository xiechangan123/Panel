package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
	Endpoint        string            // 自定义端点（如 MinIO）
	BasePath        string            // 基础路径前缀
	AddressingStyle S3AddressingStyle // 地址模式
	ForcePathStyle  bool              // 强制使用 Path 模式（兼容旧版本）
}

type S3 struct {
	client *s3.Client
	config S3Config
}

func NewS3(cfg S3Config) (Storage, error) {
	// 设置默认地址模式
	if cfg.AddressingStyle == "" {
		if cfg.ForcePathStyle {
			cfg.AddressingStyle = S3AddressingStylePath
		} else {
			cfg.AddressingStyle = S3AddressingStyleVirtualHosted
		}
	}

	cfg.BasePath = strings.Trim(cfg.BasePath, "/")

	var awsCfg aws.Config
	var err error

	if cfg.Endpoint != "" {
		// 自定义端点（如 MinIO）
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID, cfg.SecretAccessKey, "")),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           cfg.Endpoint,
						SigningRegion: cfg.Region,
					}, nil
				})),
		)
	} else {
		// 标准 AWS S3
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 根据地址模式配置客户端
	usePathStyle := cfg.AddressingStyle == S3AddressingStylePath || cfg.ForcePathStyle
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})

	s := &S3{
		client: client,
		config: cfg,
	}

	if s.config.BasePath != "" {
		if err := s.ensureBasePath(); err != nil {
			return nil, fmt.Errorf("failed to ensure base path: %w", err)
		}
	}

	return s, nil
}

// ensureBasePath 确保基础路径存在
func (s *S3) ensureBasePath() error {
	key := s.config.BasePath + "/"
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.config.BasePath),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte{}),
	})
	return err
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

// MakeDirectory 创建目录（S3中实际创建一个空的目录标记对象）
func (s *S3) MakeDirectory(directory string) error {
	key := s.getKey(directory)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte{}),
	})

	return err
}

// DeleteDirectory 删除目录
func (s *S3) DeleteDirectory(directory string) error {
	prefix := s.getKey(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 列出所有文件
	var objects []types.ObjectIdentifier
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}

		for _, obj := range output.Contents {
			if obj.Key != nil {
				objects = append(objects, types.ObjectIdentifier{
					Key: obj.Key,
				})
			}
		}
	}

	if len(objects) == 0 {
		return nil
	}

	// 批量删除
	_, err := s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(s.config.Bucket),
		Delete: &types.Delete{
			Objects: objects,
		},
	})

	return err
}

// Copy 复制文件到新位置
func (s *S3) Copy(oldFile, newFile string) error {
	sourceKey := s.getKey(oldFile)
	destKey := s.getKey(newFile)

	_, err := s.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(s.config.Bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", s.config.Bucket, sourceKey)),
		Key:        aws.String(destKey),
	})

	return err
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

// Files 获取目录下的所有文件
func (s *S3) Files(path string) ([]string, error) {
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
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		for _, obj := range output.Contents {
			if obj.Key != nil && !strings.HasSuffix(*obj.Key, "/") {
				fileName := strings.TrimPrefix(*obj.Key, prefix)
				if fileName != "" && !strings.Contains(fileName, "/") {
					files = append(files, fileName)
				}
			}
		}
	}

	return files, nil
}

// Get 读取文件内容
func (s *S3) Get(file string) ([]byte, error) {
	key := s.getKey(file)
	output, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
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

// MimeType 获取文件的 MIME 类型
func (s *S3) MimeType(file string) (string, error) {
	key := s.getKey(file)
	output, err := s.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}

	if output.ContentType != nil {
		return *output.ContentType, nil
	}

	// 根据文件扩展名推断
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream", nil
	}
	return mimeType, nil
}

// Missing 检查文件是否不存在
func (s *S3) Missing(file string) bool {
	return !s.Exists(file)
}

// Move 移动文件到新位置
func (s *S3) Move(oldFile, newFile string) error {
	// 先复制
	if err := s.Copy(oldFile, newFile); err != nil {
		return err
	}
	// 再删除原文件
	return s.Delete(oldFile)
}

// Path 获取文件的完整路径
func (s *S3) Path(file string) string {
	// 根据地址模式返回不同的 URL 格式
	key := s.getKey(file)

	if s.config.Endpoint != "" {
		// 自定义端点
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(s.config.Endpoint, "/"), s.config.Bucket, key)
	}

	switch s.config.AddressingStyle {
	case S3AddressingStyleVirtualHosted:
		// Virtual Hosted 模式：https://bucket.s3.region.amazonaws.com/key
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.config.Bucket, s.config.Region, key)
	case S3AddressingStylePath:
		// Path 模式：https://s3.region.amazonaws.com/bucket/key
		return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", s.config.Region, s.config.Bucket, key)
	default:
		// 默认返回 s3:// 协议格式
		return fmt.Sprintf("s3://%s/%s", s.config.Bucket, key)
	}
}

// Put 写入文件内容
func (s *S3) Put(file, content string) error {
	key := s.getKey(file)

	// 推断 MIME 类型
	ext := filepath.Ext(file)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: aws.String(contentType),
	})

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

	if output.ContentLength != nil {
		return *output.ContentLength, nil
	}
	return 0, nil
}
