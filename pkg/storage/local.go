package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pkgio "github.com/acepanel/panel/pkg/io"
	"github.com/shirou/gopsutil/v4/disk"
)

type Local struct {
	basePath string
}

func NewLocal(basePath string) (Storage, error) {
	if basePath == "" {
		return nil, errors.New("base path is empty")
	}
	return &Local{
		basePath: basePath,
	}, nil
}

// Delete 删除文件
func (l *Local) Delete(files ...string) error {
	for _, file := range files {
		fullPath := l.fullPath(file)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (l *Local) Exists(file string) bool {
	fullPath := l.fullPath(file)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

// LastModified 获取文件最后修改时间
func (l *Local) LastModified(file string) (time.Time, error) {
	fullPath := l.fullPath(file)
	info, err := os.Stat(fullPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// List 列出目录下的所有文件
func (l *Local) List(path string) ([]string, error) {
	fullPath := l.fullPath(path)
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

// Put 写入文件内容
func (l *Local) Put(file string, content io.Reader) error {
	fullPath := l.fullPath(file)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// 预检查空间
	if err := l.preCheckPath(fullPath); err != nil {
		return fmt.Errorf("pre check path failed: %w", err)
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	defer func(f *os.File) { _ = f.Close() }(f)

	_, err = io.Copy(f, content)
	return err
}

// Size 获取文件大小
func (l *Local) Size(file string) (int64, error) {
	fullPath := l.fullPath(file)
	info, err := os.Stat(fullPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (l *Local) fullPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return l.basePath
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.basePath, path)
}

func (l *Local) preCheckPath(path string) error {
	size, err := pkgio.SizeX(path)
	if err != nil {
		return err
	}
	files, err := pkgio.CountX(path)
	if err != nil {
		return err
	}

	usage, err := disk.Usage(l.basePath)
	if err != nil {
		return err
	}

	if uint64(size) > usage.Free {
		return errors.New("insufficient backup directory space")
	}
	if uint64(files) > usage.InodesFree {
		return errors.New("insufficient backup directory inode")
	}

	return nil
}
