package storage

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rhnvrm/simples3"
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
	AccessKey       string            // 访问密钥 ID
	SecretKey       string            // 访问密钥
	Endpoint        string            // 自定义端点
	Scheme          string            // 协议 http 或 https
	BasePath        string            // 基础路径前缀
	AddressingStyle S3AddressingStyle // 地址模式
}

type S3 struct {
	client *simples3.S3
	config S3Config
	bucket string // bucket 用于 API 调用
}

func NewS3(cfg S3Config) (Storage, error) {
	if cfg.AddressingStyle == "" {
		cfg.AddressingStyle = S3AddressingStyleVirtualHosted
	}
	if cfg.Scheme == "" {
		cfg.Scheme = "https"
	}

	cfg.BasePath = strings.Trim(cfg.BasePath, "/")

	client := simples3.New(cfg.Region, cfg.AccessKey, cfg.SecretKey)

	// bucket 用于 API 调用
	bucket := cfg.Bucket

	if cfg.Endpoint != "" {
		// 自定义 Endpoint
		if cfg.AddressingStyle == S3AddressingStyleVirtualHosted {
			// Virtual Hosted Style: https://{bucket}.{endpoint}/{key}
			client.SetEndpoint(fmt.Sprintf("%s://%s.%s", cfg.Scheme, cfg.Bucket, cfg.Endpoint))
			client.SetVirtualHostedStyle()
			bucket = ""
		} else {
			// Path Style: https://{endpoint}/{bucket}/{key}
			client.SetEndpoint(fmt.Sprintf("%s://%s", cfg.Scheme, cfg.Endpoint))
		}
	} else {
		// AWS S3
		if cfg.AddressingStyle == S3AddressingStyleVirtualHosted {
			// Virtual Hosted Style: https://{bucket}.s3.{region}.amazonaws.com/{key}
			client.SetEndpoint(fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region))
			client.SetVirtualHostedStyle()
			bucket = ""
		}
	}

	return &S3{
		client: client,
		config: cfg,
		bucket: bucket,
	}, nil
}

// Delete 删除文件
func (s *S3) Delete(files ...string) error {
	if len(files) == 0 {
		return nil
	}

	// 批量删除
	var objects []string
	for _, file := range files {
		key := s.getKey(file)
		objects = append(objects, key)
	}

	_, err := s.client.DeleteObjects(simples3.DeleteObjectsInput{
		Bucket:  s.bucket,
		Objects: objects,
		Quiet:   true,
	})

	return err
}

// Exists 检查文件是否存在
func (s *S3) Exists(file string) bool {
	key := s.getKey(file)
	_, err := s.client.FileDetails(simples3.DetailsInput{
		Bucket:    s.bucket,
		ObjectKey: key,
	})
	return err == nil
}

// LastModified 获取文件最后修改时间
func (s *S3) LastModified(file string) (time.Time, error) {
	key := s.getKey(file)
	output, err := s.client.FileDetails(simples3.DetailsInput{
		Bucket:    s.bucket,
		ObjectKey: key,
	})
	if err != nil {
		return time.Time{}, err
	}

	if output.LastModified == "" {
		return time.Time{}, nil
	}

	// 解析 HTTP 日期格式
	t, err := time.Parse(time.RFC1123, output.LastModified)
	if err != nil {
		// 尝试其他格式
		t, err = time.Parse(time.RFC1123Z, output.LastModified)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse LastModified: %w", err)
		}
	}

	return t, nil
}

// List 列出目录下的所有文件
func (s *S3) List(path string) ([]string, error) {
	prefix := s.getKey(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var files []string
	seq, finish := s.client.ListAll(simples3.ListInput{
		Bucket:    s.bucket,
		Prefix:    prefix,
		Delimiter: "/",
	})

	for obj := range seq {
		key := obj.Key
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

	if err := finish(); err != nil {
		return nil, err
	}

	return files, nil
}

// Put 写入文件内容
func (s *S3) Put(file string, content io.Reader) error {
	key := s.getKey(file)

	_, err := s.client.FileUploadMultipart(simples3.MultipartUploadInput{
		Bucket:      s.bucket,
		ObjectKey:   key,
		ContentType: "application/octet-stream",
		Body:        content,
		Concurrency: 5,
	})

	return err
}

// Size 获取文件大小
func (s *S3) Size(file string) (int64, error) {
	key := s.getKey(file)
	output, err := s.client.FileDetails(simples3.DetailsInput{
		Bucket:    s.bucket,
		ObjectKey: key,
	})
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseInt(output.ContentLength, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ContentLength: %w", err)
	}

	return size, nil
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
