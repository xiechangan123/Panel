package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/acepanel/panel/v3/pkg/chattr"
)

// setImmutable 尝试为文件设置 immutable 属性，返回是否设置成功
// 需要 root 权限且文件系统支持，否则返回 false 用于跳过测试
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

// isImmutable 判断文件是否设置了 immutable 属性
func isImmutable(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	ok, _ := chattr.IsAttr(file, chattr.FS_IMMUTABLE_FL)
	return ok
}

// TestChmodNormalDir 无锁定文件时 Chmod 应递归生效，验证 withUnlock 正常路径不影响行为
func TestChmodNormalDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_chmod_normal")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sub := filepath.Join(tmpDir, "sub")
	assert.NoError(t, os.Mkdir(sub, 0755))
	file := filepath.Join(sub, "f.txt")
	assert.NoError(t, Write(file, "x", 0644))

	assert.NoError(t, Chmod(tmpDir, 0700))

	// 递归应作用到子目录下的文件
	info, err := os.Stat(file)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

// TestUnlockRelockAttr 解锁应仅记录被锁定的文件，且能原样恢复
func TestUnlockRelockAttr(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_unlock_attr")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	locked := filepath.Join(tmpDir, "locked.txt")
	normal := filepath.Join(tmpDir, "normal.txt")
	assert.NoError(t, Write(locked, "a", 0644))
	assert.NoError(t, Write(normal, "b", 0644))

	if !setImmutable(locked) {
		t.Skip("当前环境不支持设置 immutable 属性（需 root 与支持的文件系统）")
	}
	defer unlockAttr(tmpDir) // 确保用例结束后目录可被清理

	// 解锁后只应记录被锁定文件，且属性已解除
	files := unlockAttr(tmpDir)
	assert.Len(t, files, 1)
	assert.Equal(t, locked, files[0].path)
	assert.False(t, isImmutable(locked))

	// 恢复后被锁定文件应重新带上 immutable
	relockAttr(files)
	assert.True(t, isImmutable(locked))
}

// TestChmodWithImmutable 目录下存在 immutable 文件时 Chmod 仍应成功，并在完成后恢复锁定
func TestChmodWithImmutable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_chmod_immutable")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	userIni := filepath.Join(tmpDir, ".user.ini")
	assert.NoError(t, Write(userIni, "open_basedir=/tmp/", 0644))

	if !setImmutable(userIni) {
		t.Skip("当前环境不支持设置 immutable 属性（需 root 与支持的文件系统）")
	}
	defer unlockAttr(tmpDir)

	// 递归 chmod 会被 immutable 文件阻挡，withUnlock 应解锁重试并成功
	assert.NoError(t, Chmod(tmpDir, 0700))

	info, err := os.Stat(tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	// immutable 保护应已恢复
	assert.True(t, isImmutable(userIni))
}
