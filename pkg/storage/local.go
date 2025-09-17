package storage

import (
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

type Local struct {
	basePath string
}

func NewLocal(basePath string) Storage {
	if basePath == "" {
		basePath = "/"
	}
	return &Local{
		basePath: basePath,
	}
}

// MakeDirectory 创建目录
func (n *Local) MakeDirectory(directory string) error {
	fullPath := n.fullPath(directory)
	return os.MkdirAll(fullPath, 0755)
}

// DeleteDirectory 删除目录
func (n *Local) DeleteDirectory(directory string) error {
	fullPath := n.fullPath(directory)
	return os.RemoveAll(fullPath)
}

// Copy 复制文件到新位置
func (n *Local) Copy(oldFile, newFile string) error {
	srcPath := n.fullPath(oldFile)
	dstPath := n.fullPath(newFile)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	_, err = io.Copy(dst, src)
	return err
}

// Delete 删除文件
func (n *Local) Delete(files ...string) error {
	for _, file := range files {
		fullPath := n.fullPath(file)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (n *Local) Exists(file string) bool {
	fullPath := n.fullPath(file)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

// Files 获取目录下的所有文件
func (n *Local) Files(path string) ([]string, error) {
	fullPath := n.fullPath(path)
	entries, err := os.ReadDir(fullPath)
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
func (n *Local) Get(file string) ([]byte, error) {
	fullPath := n.fullPath(file)
	return os.ReadFile(fullPath)
}

// LastModified 获取文件最后修改时间
func (n *Local) LastModified(file string) (time.Time, error) {
	fullPath := n.fullPath(file)
	info, err := os.Stat(fullPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// MimeType 获取文件的 MIME 类型
func (n *Local) MimeType(file string) (string, error) {
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream", nil
	}
	return mimeType, nil
}

// Missing 检查文件是否不存在
func (n *Local) Missing(file string) bool {
	return !n.Exists(file)
}

// Move 移动文件到新位置
func (n *Local) Move(oldFile, newFile string) error {
	oldPath := n.fullPath(oldFile)
	newPath := n.fullPath(newFile)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		return err
	}

	return os.Rename(oldPath, newPath)
}

// Path 获取文件的完整路径
func (n *Local) Path(file string) string {
	return n.fullPath(file)
}

// Put 写入文件内容
func (n *Local) Put(file, content string) error {
	fullPath := n.fullPath(file)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// Size 获取文件大小
func (n *Local) Size(file string) (int64, error) {
	fullPath := n.fullPath(file)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// fullPath 获取文件的完整路径
func (n *Local) fullPath(file string) string {
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(n.basePath, file)
}
