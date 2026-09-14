package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"

	"github.com/acepanel/panel/v3/pkg/chattr"
)

// setImmutable 设置 immutable，需 root 且文件系统支持，失败返回 false 用于跳过用例
func setImmutable(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	if err = chattr.SetAttr(file, chattr.FS_IMMUTABLE_FL); err != nil {
		return false
	}
	ok, _ := chattr.IsAttr(file, chattr.FS_IMMUTABLE_FL)
	return ok
}

func isImmutable(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	ok, _ := chattr.IsAttr(file, chattr.FS_IMMUTABLE_FL)
	return ok
}

func TestChmodNormalDir(t *testing.T) {
	tmpDir := t.TempDir()

	sub := filepath.Join(tmpDir, "sub")
	must.NoError(t, os.Mkdir(sub, 0755))
	file := filepath.Join(sub, "f.txt")
	must.NoError(t, Write(file, "x", 0644))

	check.NoError(t, Chmod(t.Context(), tmpDir, 0700))

	info, err := os.Stat(file)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0700))
}

func TestUnlockRelockAttr(t *testing.T) {
	tmpDir := t.TempDir()

	locked := filepath.Join(tmpDir, "locked.txt")
	normal := filepath.Join(tmpDir, "normal.txt")
	must.NoError(t, Write(locked, "a", 0644))
	must.NoError(t, Write(normal, "b", 0644))

	if !setImmutable(locked) {
		t.Skip("当前环境不支持设置 immutable 属性（需 root 与支持的文件系统）")
	}
	defer unlockAttr(tmpDir) // 确保用例结束后目录可被清理

	files := unlockAttr(tmpDir)
	must.Len(t, files, 1)
	check.Equal(t, files[0].path, locked)
	check.False(t, isImmutable(locked), check.Msgf("应已解除 immutable: %s", locked))

	relockAttr(files)
	check.True(t, isImmutable(locked), check.Msgf("应已恢复 immutable: %s", locked))
}

func TestChmodWithImmutable(t *testing.T) {
	tmpDir := t.TempDir()

	userIni := filepath.Join(tmpDir, ".user.ini")
	must.NoError(t, Write(userIni, "open_basedir=/tmp/", 0644))

	if !setImmutable(userIni) {
		t.Skip("当前环境不支持设置 immutable 属性（需 root 与支持的文件系统）")
	}
	defer unlockAttr(tmpDir)

	// 递归 chmod 会被 immutable 阻挡，withUnlock 应解锁重试
	check.NoError(t, Chmod(t.Context(), tmpDir, 0700))

	info, err := os.Stat(tmpDir)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0700))

	check.True(t, isImmutable(userIni), check.Msgf("应已恢复 immutable: %s", userIni))
}
