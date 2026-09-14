package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/shirou/gopsutil/v4/disk"

	pkgio "github.com/acepanel/panel/v3/pkg/io"
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
func (l *Local) Delete(_ context.Context, files ...string) error {
	for _, file := range files {
		fullPath := l.fullPath(file)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (l *Local) Exists(_ context.Context, file string) bool {
	fullPath := l.fullPath(file)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}

// LastModified 获取文件最后修改时间
func (l *Local) LastModified(_ context.Context, file string) (time.Time, error) {
	fullPath := l.fullPath(file)
	info, err := os.Stat(fullPath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// List 列出目录下的所有文件
func (l *Local) List(_ context.Context, path string) ([]string, error) {
	fullPath := l.fullPath(path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	files := lo.FilterMap(entries, func(entry os.DirEntry, _ int) (string, bool) {
		if entry.IsDir() {
			return "", false
		}
		return entry.Name(), true
	})
	return files, nil
}

// Put 写入文件内容
func (l *Local) Put(ctx context.Context, file string, content io.Reader) error {
	fullPath := l.fullPath(file)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// 预检查空间
	if err := l.preCheckPath(ctx, filepath.Dir(fullPath)); err != nil {
		return fmt.Errorf("pre check path failed: %w", err)
	}

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, &ctxReader{ctx: ctx, r: content}); err != nil {
		_ = f.Close()
		return err
	}

	// 写入错误可能要到 Close 才暴露（如磁盘写满），这里不能吞
	return f.Close()
}

// Rename 改名
func (l *Local) Rename(_ context.Context, src, dst string) error {
	return os.Rename(l.fullPath(src), l.fullPath(dst))
}

// Size 获取文件大小
func (l *Local) Size(_ context.Context, file string) (int64, error) {
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

func (l *Local) preCheckPath(ctx context.Context, path string) error {
	size, err := pkgio.SizeX(ctx, path)
	if err != nil {
		return err
	}
	files, err := pkgio.CountX(ctx, path)
	if err != nil {
		return err
	}

	usage, err := disk.Usage(l.basePath)
	if err != nil {
		return err
	}

	if size > 0 && uint64(size) > usage.Free {
		return errors.New("insufficient backup directory space")
	}
	if files > 0 && uint64(files) > usage.InodesFree {
		return errors.New("insufficient backup directory inode")
	}

	return nil
}
