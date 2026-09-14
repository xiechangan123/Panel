package io

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/acepanel/panel/v3/pkg/chattr"
	"github.com/acepanel/panel/v3/pkg/shell"
)

// Remove 删除文件/目录
// 不响应取消：RemoveAll 无法中途停下，解锁被取消只会让删除撞上 immutable 文件半途失败
func Remove(ctx context.Context, path string) error {
	_, _ = shell.Execf(context.WithoutCancel(ctx), "chattr -R -ia '%s'", path)
	return os.RemoveAll(path)
}

// Chmod 修改文件/目录权限
func Chmod(ctx context.Context, path string, permission os.FileMode) error {
	return withUnlock(path, func() (string, error) {
		return shell.Execf(ctx, "chmod -R '%o' '%s'", permission, path)
	})
}

// Chown 修改文件或目录所有者
func Chown(ctx context.Context, path, user, group string) error {
	return withUnlock(path, func() (string, error) {
		return shell.Execf(ctx, "chown -R '%s:%s' '%s'", user, group, path)
	})
}

// withUnlock 执行 chmod/chown 等递归命令，若被 immutable/append 锁定文件（如 .user.ini）阻挡，
// 则解锁后重试并在完成后恢复锁定属性
func withUnlock(path string, run func() (string, error)) error {
	_, err := run()
	if err == nil || !strings.Contains(err.Error(), "Operation not permitted") {
		return err
	}
	locked := unlockAttr(path)
	_, err = run()
	relockAttr(locked)
	return err
}

type lockedFile struct {
	path  string
	attrs uint32
}

// unlockAttr 递归解除 path 下文件的 immutable/append 属性，返回解除记录用于后续恢复
func unlockAttr(path string) []lockedFile {
	var locked []lockedFile
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr
		}
		file, err := os.OpenFile(p, os.O_RDONLY, 0) //nolint:gosec
		if err != nil {
			return nil //nolint:nilerr
		}
		var attrs uint32
		if ok, _ := chattr.IsAttr(file, chattr.FS_IMMUTABLE_FL); ok {
			attrs |= chattr.FS_IMMUTABLE_FL
		}
		if ok, _ := chattr.IsAttr(file, chattr.FS_APPEND_FL); ok {
			attrs |= chattr.FS_APPEND_FL
		}
		if attrs != 0 {
			_ = chattr.UnsetAttr(file, attrs)
			locked = append(locked, lockedFile{path: p, attrs: attrs})
		}
		_ = file.Close()
		return nil
	})
	return locked
}

// relockAttr 恢复 unlockAttr 解除的 immutable/append 属性
func relockAttr(files []lockedFile) {
	for _, f := range files {
		file, err := os.OpenFile(f.path, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		_ = chattr.SetAttr(file, f.attrs)
		_ = file.Close()
	}
}

// Exists 判断路径是否存在
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Empty 判断路径是否为空
func Empty(path string) bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return true
	}

	return len(files) == 0
}

func Mv(ctx context.Context, src, dst string) error {
	_, err := shell.Execf(ctx, `mv -f '%s' '%s'`, src, dst)
	return err
}

// Cp 复制文件或目录（保留所有权和权限）
func Cp(ctx context.Context, src, dst string) error {
	_, err := shell.Execf(ctx, `cp -arf '%s' '%s'`, src, dst)
	return err
}

// Size 获取路径大小
func Size(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		size += info.Size()
		return nil
	})

	return size, err
}

// IsDir 判断是否为目录
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// SizeX 获取路径大小（du命令）
func SizeX(ctx context.Context, path string) (int64, error) {
	out, err := shell.Execf(ctx, "du -sb '%s'", path)
	if err != nil {
		return 0, err
	}

	parts := strings.Fields(out)
	if len(parts) == 0 {
		return 0, fmt.Errorf("failed to parse du output: %s", out)
	}

	return strconv.ParseInt(parts[0], 10, 64)
}

// CountX 统计目录下文件数
func CountX(ctx context.Context, path string) (int64, error) {
	out, err := shell.Execf(ctx, "find '%s' -printf '.'", path)
	if err != nil {
		return 0, err
	}

	count := len(out)
	return int64(count), nil
}
