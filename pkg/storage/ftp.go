package storage

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type FTPConfig struct {
	Host     string // FTP 服务器地址
	Port     int    // FTP 端口，默认 21
	Username string // 用户名
	Password string // 密码
	BasePath string // 基础路径
}

type FTP struct {
	config FTPConfig
}

func NewFTP(config FTPConfig) (Storage, error) {
	if config.Port == 0 {
		config.Port = 21
	}
	config.BasePath = strings.Trim(config.BasePath, "/")

	f := &FTP{
		config: config,
	}

	if err := f.ensureBasePath(); err != nil {
		return nil, fmt.Errorf("failed to ensure base path: %w", err)
	}

	return f, nil
}

// connect 建立 FTP 连接
func (f *FTP) connect() (*ftp.ServerConn, error) {
	addr := fmt.Sprintf("%s:%d", f.config.Host, f.config.Port)
	conn, err := ftp.Dial(addr)
	if err != nil {
		return nil, err
	}

	err = conn.Login(f.config.Username, f.config.Password)
	if err != nil {
		conn.Quit()
		return nil, err
	}

	return conn, nil
}

// ensureBasePath 确保基础路径存在
func (f *FTP) ensureBasePath() error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	// 递归创建路径
	parts := strings.Split(f.config.BasePath, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		_ = conn.MakeDir(currentPath)
	}

	return nil
}

// getRemotePath 获取远程路径
func (f *FTP) getRemotePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if f.config.BasePath == "" {
		return path
	}
	if path == "" {
		return f.config.BasePath
	}
	return fmt.Sprintf("%s/%s", f.config.BasePath, path)
}

// MakeDirectory 创建目录
func (f *FTP) MakeDirectory(directory string) error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(directory)

	// 递归创建目录
	parts := strings.Split(remotePath, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// 尝试创建目录
		_ = conn.MakeDir(currentPath)
	}

	return nil
}

// DeleteDirectory 删除目录
func (f *FTP) DeleteDirectory(directory string) error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(directory)
	return conn.RemoveDir(remotePath)
}

// Copy 复制文件到新位置
func (f *FTP) Copy(oldFile, newFile string) error {
	// FTP 不支持直接复制，需要下载再上传
	data, err := f.Get(oldFile)
	if err != nil {
		return err
	}
	return f.Put(newFile, string(data))
}

// Delete 删除文件
func (f *FTP) Delete(files ...string) error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	for _, file := range files {
		remotePath := f.getRemotePath(file)
		if err := conn.Delete(remotePath); err != nil {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (f *FTP) Exists(file string) bool {
	conn, err := f.connect()
	if err != nil {
		return false
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(file)
	_, err = conn.FileSize(remotePath)
	return err == nil
}

// Files 获取目录下的所有文件
func (f *FTP) Files(path string) ([]string, error) {
	conn, err := f.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(path)
	entries, err := conn.List(remotePath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.Type == ftp.EntryTypeFile {
			files = append(files, entry.Name)
		}
	}

	return files, nil
}

// Get 读取文件内容
func (f *FTP) Get(file string) ([]byte, error) {
	conn, err := f.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(file)
	resp, err := conn.Retr(remotePath)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	return io.ReadAll(resp)
}

// LastModified 获取文件最后修改时间
func (f *FTP) LastModified(file string) (time.Time, error) {
	conn, err := f.connect()
	if err != nil {
		return time.Time{}, err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(file)
	entries, err := conn.List(filepath.Dir(remotePath))
	if err != nil {
		return time.Time{}, err
	}

	fileName := filepath.Base(remotePath)
	for _, entry := range entries {
		if entry.Name == fileName {
			return entry.Time, nil
		}
	}

	return time.Time{}, fmt.Errorf("file not found: %s", file)
}

// MimeType 获取文件的 MIME 类型
func (f *FTP) MimeType(file string) (string, error) {
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream", nil
	}
	return mimeType, nil
}

// Missing 检查文件是否不存在
func (f *FTP) Missing(file string) bool {
	return !f.Exists(file)
}

// Move 移动文件到新位置
func (f *FTP) Move(oldFile, newFile string) error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	oldPath := f.getRemotePath(oldFile)
	newPath := f.getRemotePath(newFile)

	// 确保目标目录存在
	newDir := filepath.Dir(newPath)
	if newDir != "." {
		f.createDirectoryPath(conn, newDir)
	}

	return conn.Rename(oldPath, newPath)
}

// createDirectoryPath 递归创建目录路径
func (f *FTP) createDirectoryPath(conn *ftp.ServerConn, path string) {
	parts := strings.Split(path, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		_ = conn.MakeDir(currentPath)
	}
}

// Path 获取文件的完整路径
func (f *FTP) Path(file string) string {
	return fmt.Sprintf("ftp://%s:%d/%s", f.config.Host, f.config.Port, f.getRemotePath(file))
}

// Put 写入文件内容
func (f *FTP) Put(file, content string) error {
	conn, err := f.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(file)

	// 确保目录存在
	remoteDir := filepath.Dir(remotePath)
	if remoteDir != "." {
		f.createDirectoryPath(conn, remoteDir)
	}

	return conn.Stor(remotePath, bytes.NewReader([]byte(content)))
}

// Size 获取文件大小
func (f *FTP) Size(file string) (int64, error) {
	conn, err := f.connect()
	if err != nil {
		return 0, err
	}
	defer conn.Quit()

	remotePath := f.getRemotePath(file)
	return conn.FileSize(remotePath)
}
