package storage

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/pkg/storage/s3sdk"
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
	client *s3sdk.S3
	config S3Config
}

func NewS3(cfg S3Config) (Storage, error) {
	cfg.BasePath = strings.Trim(cfg.BasePath, "/")

	client := s3sdk.New(s3sdk.Config{
		Region:      cfg.Region,
		Bucket:      cfg.Bucket,
		AccessKey:   cfg.AccessKey,
		SecretKey:   cfg.SecretKey,
		Endpoint:    cfg.Endpoint,
		Scheme:      cfg.Scheme,
		PathStyle:   cfg.AddressingStyle == S3AddressingStylePath,
		Concurrency: 5,
	})

	return &S3{client: client, config: cfg}, nil
}

// Delete 删除文件
func (s *S3) Delete(files ...string) error {
	if len(files) == 0 {
		return nil
	}
	keys := lo.Map(files, func(file string, _ int) string {
		return s.getKey(file)
	})
	return s.client.Delete(keys...)
}

// Exists 检查文件是否存在
func (s *S3) Exists(file string) bool {
	_, err := s.client.Stat(s.getKey(file))
	return err == nil
}

// LastModified 获取文件最后修改时间
func (s *S3) LastModified(file string) (time.Time, error) {
	info, err := s.client.Stat(s.getKey(file))
	if err != nil {
		return time.Time{}, err
	}
	return info.LastModified, nil
}

// List 列出目录下的所有文件
func (s *S3) List(path string) ([]string, error) {
	prefix := s.getKey(path)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var files []string
	for obj, err := range s.client.List(prefix, "/") {
		if err != nil {
			return nil, err
		}
		// 提取文件名，跳过子目录
		name := strings.TrimPrefix(obj.Key, prefix)
		if name != "" && !strings.Contains(name, "/") {
			files = append(files, name)
		}
	}

	return files, nil
}

// Put 写入文件内容
func (s *S3) Put(file string, content io.Reader) error {
	return s.client.Put(s.getKey(file), content, "application/octet-stream")
}

// Size 获取文件大小
func (s *S3) Size(file string) (int64, error) {
	info, err := s.client.Stat(s.getKey(file))
	if err != nil {
		return 0, err
	}
	return info.Size, nil
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
