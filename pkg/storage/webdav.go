package storage

import (
	"bytes"
	"fmt"
	"io"
	"mime"
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

	w := &WebDav{
		client: client,
		config: config,
	}

	if err := w.ensureBasePath(); err != nil {
		return nil, fmt.Errorf("failed to ensure base path: %w", err)
	}

	return w, nil
}

// ensureBasePath 确保基础路径存在
func (w *WebDav) ensureBasePath() error {
	if w.config.BasePath == "" {
		return nil
	}

	return w.client.MkdirAll(w.config.BasePath, 0755)
}

// getRemotePath 获取远程路径
func (w *WebDav) getRemotePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if w.config.BasePath == "" {
		return path
	}
	if path == "" {
		return w.config.BasePath
	}
	return filepath.Join(w.config.BasePath, path)
}

// MakeDirectory 创建目录
func (w *WebDav) MakeDirectory(directory string) error {
	remotePath := w.getRemotePath(directory)
	return w.client.MkdirAll(remotePath, 0755)
}

// DeleteDirectory 删除目录
func (w *WebDav) DeleteDirectory(directory string) error {
	remotePath := w.getRemotePath(directory)
	return w.client.RemoveAll(remotePath)
}

// Copy 复制文件到新位置
func (w *WebDav) Copy(oldFile, newFile string) error {
	oldPath := w.getRemotePath(oldFile)
	newPath := w.getRemotePath(newFile)

	// 确保目标目录存在
	newDir := filepath.Dir(newPath)
	if newDir != "." {
		_ = w.client.MkdirAll(newDir, 0755)
	}

	return w.client.Copy(oldPath, newPath, false)
}

// Delete 删除文件
func (w *WebDav) Delete(files ...string) error {
	for _, file := range files {
		remotePath := w.getRemotePath(file)
		if err := w.client.Remove(remotePath); err != nil {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (w *WebDav) Exists(file string) bool {
	remotePath := w.getRemotePath(file)
	_, err := w.client.Stat(remotePath)
	return err == nil
}

// Files 获取目录下的所有文件
func (w *WebDav) Files(path string) ([]string, error) {
	remotePath := w.getRemotePath(path)
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

// Get 读取文件内容
func (w *WebDav) Get(file string) ([]byte, error) {
	remotePath := w.getRemotePath(file)
	reader, err := w.client.ReadStream(remotePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

// LastModified 获取文件最后修改时间
func (w *WebDav) LastModified(file string) (time.Time, error) {
	remotePath := w.getRemotePath(file)
	stat, err := w.client.Stat(remotePath)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

// MimeType 获取文件的 MIME 类型
func (w *WebDav) MimeType(file string) (string, error) {
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream", nil
	}
	return mimeType, nil
}

// Missing 检查文件是否不存在
func (w *WebDav) Missing(file string) bool {
	return !w.Exists(file)
}

// Move 移动文件到新位置
func (w *WebDav) Move(oldFile, newFile string) error {
	oldPath := w.getRemotePath(oldFile)
	newPath := w.getRemotePath(newFile)

	// 确保目标目录存在
	newDir := filepath.Dir(newPath)
	if newDir != "." {
		_ = w.client.MkdirAll(newDir, 0755)
	}

	return w.client.Rename(oldPath, newPath, false)
}

// Path 获取文件的完整路径
func (w *WebDav) Path(file string) string {
	remotePath := w.getRemotePath(file)
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(w.config.URL, "/"), remotePath)
}

// Put 写入文件内容
func (w *WebDav) Put(file, content string) error {
	remotePath := w.getRemotePath(file)

	// 确保目录存在
	remoteDir := filepath.Dir(remotePath)
	if remoteDir != "." {
		_ = w.client.MkdirAll(remoteDir, 0755)
	}

	return w.client.WriteStream(remotePath, bytes.NewReader([]byte(content)), 0644)
}

// Size 获取文件大小
func (w *WebDav) Size(file string) (int64, error) {
	remotePath := w.getRemotePath(file)
	stat, err := w.client.Stat(remotePath)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}
