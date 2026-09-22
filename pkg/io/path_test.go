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

// requireImmutable 不支持时跳过用例
func requireImmutable(t *testing.T, path string) {
	t.Helper()
	if !setImmutable(path) {
		t.Skip("当前环境不支持设置 immutable 属性（需 root 与支持的文件系统）")
	}
	t.Cleanup(func() { unlockTree(filepath.Dir(path)) })
}

func TestChmodROnDir(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "sub", "f.txt")
	must.NoError(t, Write(file, "x", 0644))

	check.NoError(t, ChmodR(t.Context(), tmpDir, 0700))

	info, err := os.Stat(file)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0700))
}

func TestChmodOnlyTouchesItself(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "f.txt")
	must.NoError(t, Write(file, "x", 0644))

	check.NoError(t, Chmod(tmpDir, 0700))

	info, err := os.Stat(file)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0644))
}

func TestChmodFollowsRootSymlink(t *testing.T) {
	dir := t.TempDir()
	target, link := filepath.Join(dir, "real"), filepath.Join(dir, "link")
	must.NoError(t, Write(filepath.Join(target, "f.txt"), "x", 0644))
	must.NoError(t, os.Symlink(target, link))

	check.NoError(t, Chmod(link, 0700))
	info, err := os.Stat(target)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0700))

	check.NoError(t, ChmodR(t.Context(), link, 0600))
	info, err = os.Stat(filepath.Join(target, "f.txt"))
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0600))
}

func TestChmodKeepsSpecialBits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.txt")
	must.NoError(t, Write(file, "x", 0644))

	check.NoError(t, Chmod(file, 0o4755))

	info, err := os.Stat(file)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0755))
	check.True(t, info.Mode()&os.ModeSetuid != 0, check.Msgf("setuid 位应保留"))
}

func TestUnlockTreeCoversDirsAndFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sub := filepath.Join(tmpDir, "sub")
	locked := filepath.Join(sub, "locked.txt")
	must.NoError(t, Write(locked, "a", 0644))
	must.NoError(t, Write(filepath.Join(sub, "normal.txt"), "b", 0644))
	requireImmutable(t, locked)
	if !setImmutable(sub) {
		t.Skip("目录不支持 immutable")
	}

	files := unlockTree(tmpDir)
	must.Len(t, files, 2)
	check.False(t, isImmutable(locked), check.Msgf("应已解除 immutable: %s", locked))
	check.False(t, isImmutable(sub), check.Msgf("应已解除目录 immutable: %s", sub))

	relockAttr(files)
	check.True(t, isImmutable(locked), check.Msgf("应已恢复 immutable: %s", locked))
	check.True(t, isImmutable(sub), check.Msgf("应已恢复目录 immutable: %s", sub))
}

func TestChmodRWithImmutable(t *testing.T) {
	tmpDir := t.TempDir()
	userIni := filepath.Join(tmpDir, ".user.ini")
	must.NoError(t, Write(userIni, "open_basedir=/tmp/", 0644))
	requireImmutable(t, userIni)

	check.NoError(t, ChmodR(t.Context(), tmpDir, 0700))

	info, err := os.Stat(userIni)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0700))
	check.True(t, isImmutable(userIni), check.Msgf("应已恢复 immutable: %s", userIni))
}

func TestRemoveWithImmutable(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "site")
	userIni := filepath.Join(tmpDir, "public", ".user.ini")
	must.NoError(t, Write(userIni, "x", 0644))
	must.NoError(t, Write(filepath.Join(tmpDir, "index.php"), "x", 0644))
	requireImmutable(t, userIni)

	check.NoError(t, Remove(tmpDir))
	check.False(t, Exists(tmpDir), check.Msgf("目录应已删除: %s", tmpDir))
}

func TestWriteKeepsImmutable(t *testing.T) {
	userIni := filepath.Join(t.TempDir(), ".user.ini")
	must.NoError(t, Write(userIni, "old", 0644))
	requireImmutable(t, userIni)

	check.NoError(t, Write(userIni, "new", 0644))
	data, err := Read(userIni)
	check.NoError(t, err)
	check.Equal(t, data, "new")
	check.True(t, isImmutable(userIni), check.Msgf("写入后应保持 immutable: %s", userIni))
}

func TestMvMergeReplacesImmutable(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src", "public"), filepath.Join(dir, "dst", "public")
	writeTree(t, src, map[string]string{".user.ini": "new", "index.php": "new"})
	writeTree(t, dst, map[string]string{".user.ini": "old", "index.php": "old"})
	requireImmutable(t, filepath.Join(dst, ".user.ini"))

	check.NoError(t, Mv(t.Context(), src, dst))
	check.False(t, Exists(src), check.Msgf("源目录应已移走: %s", src))
	checkTree(t, dst, map[string]string{".user.ini": "new", "index.php": "new"})
	check.True(t, isImmutable(filepath.Join(dst, ".user.ini")), check.Msgf("覆盖后目标应保持 immutable"))
}
