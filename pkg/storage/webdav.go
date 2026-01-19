package storage

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

type WebDavConfig struct {
	URL      string        // WebDAV 服务器 URL
	Username string        // 用户名
	Password string        // 密码
	BasePath string        // 基础路径
	Timeout  time.Duration // 连接超时时间
}

type WebDav struct {
	client *gowebdav.Client
	config WebDavConfig
}

func NewWebDav(config WebDavConfig) (Storage, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	config.BasePath = strings.Trim(config.BasePath, "/")

	client := gowebdav.NewClient(config.URL, config.Username, config.Password)
	client.SetTimeout(config.Timeout)

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to WebDAV server: %w", err)
	}

	w := &WebDav{
		client: client,
		config: config,
	}

	if w.config.BasePath != "" {
		if err := w.client.MkdirAll(w.config.BasePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create base path: %w", err)
		}
	}

	return w, nil
}

// Delete 删除文件
func (w *WebDav) Delete(files ...string) error {
	for _, file := range files {
		remotePath := w.fullPath(file)
		if err := w.client.Remove(remotePath); err != nil {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (w *WebDav) Exists(file string) bool {
	remotePath := w.fullPath(file)
	_, err := w.client.Stat(remotePath)
	return err == nil
}

// LastModified 获取文件最后修改时间
func (w *WebDav) LastModified(file string) (time.Time, error) {
	remotePath := w.fullPath(file)
	stat, err := w.client.Stat(remotePath)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

// Put 写入文件内容
func (w *WebDav) Put(file string, content io.Reader) error {
	remotePath := w.fullPath(file)

	// 确保目录存在
	remoteDir := filepath.Dir(remotePath)
	if remoteDir != "." {
		if err := w.client.MkdirAll(remoteDir, 0755); err != nil {
			return err
		}
	}

	// 调整超时
	w.client.SetTimeout(0)
	defer w.client.SetTimeout(w.config.Timeout)

	return w.client.WriteStream(remotePath, content, 0644)
}

// Size 获取文件大小
func (w *WebDav) Size(file string) (int64, error) {
	remotePath := w.fullPath(file)
	stat, err := w.client.Stat(remotePath)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// List 列出目录下的所有文件
func (w *WebDav) List(path string) ([]string, error) {
	remotePath := w.fullPath(path)
	entries, err := w.client.ReadDir(remotePath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func (w *WebDav) fullPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if w.config.BasePath == "" {
		return path
	}
	if path == "" {
		return w.config.BasePath
	}
	return filepath.Join(w.config.BasePath, path)
}
