package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
	"github.com/libtnb/utils/env"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func TestWriteCreatesFileWithCorrectContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "write_test.txt")
	data := "Hello, World!"
	permission := os.FileMode(0644)

	must.NoError(t, Write(path, data, permission))

	content, err := Read(path)
	check.NoError(t, err)
	check.Equal(t, content, data)
}

func TestWriteAppendAppendsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "append_test.txt")
	initialData := "Hello"
	appendData := ", World!"

	must.NoError(t, Write(path, initialData, 0644))
	must.NoError(t, WriteAppend(path, appendData, 0644))

	content, err := Read(path)
	check.NoError(t, err)
	check.Equal(t, content, "Hello, World!")
}

// archiveExts 归档格式，支持多文件
var archiveExts = []string{".zip", ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tar.xz", ".tar.zst", ".7z"}

// singleExts 只能压缩单个文件的格式
var singleExts = []string{".gz", ".bz2", ".xz", ".zst"}

func TestCompress(t *testing.T) {
	abs := t.TempDir()
	src := []string{"compress_test1.txt", "compress_test2.txt"}
	must.NoError(t, Write(filepath.Join(abs, src[0]), "File 1", 0644))
	must.NoError(t, Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		t.Run(ext, func(t *testing.T) {
			check.NoError(t, Compress(abs, src, filepath.Join(abs, "compress_test"+ext)))
		})
	}
	for _, ext := range singleExts {
		t.Run(ext, func(t *testing.T) {
			check.NoError(t, Compress(abs, src[:1], filepath.Join(abs, "compress_single"+ext)))
			// 单文件格式装不下多个文件
			check.Error(t, Compress(abs, src, filepath.Join(abs, "compress_multi"+ext)))
		})
	}
}

func TestUnCompress(t *testing.T) {
	abs := t.TempDir()
	src := []string{"uncompress_test1.txt", "uncompress_test2.txt"}
	must.NoError(t, Write(filepath.Join(abs, src[0]), "File 1", 0644))
	must.NoError(t, Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		t.Run(ext, func(t *testing.T) {
			dst := filepath.Join(abs, "uncompressed"+strings.ReplaceAll(ext, ".", "_"))
			must.NoError(t, Compress(abs, src, filepath.Join(abs, "uncompress_test"+ext)))
			must.NoError(t, UnCompress(filepath.Join(abs, "uncompress_test"+ext), dst))

			data, err := Read(filepath.Join(dst, src[0]))
			check.NoError(t, err)
			check.Equal(t, data, "File 1")
			data, err = Read(filepath.Join(dst, src[1]))
			check.NoError(t, err)
			check.Equal(t, data, "File 2")
		})
	}
	// 单文件压缩格式解压后去掉压缩后缀恢复原文件名
	for _, ext := range singleExts {
		t.Run(ext, func(t *testing.T) {
			dst := filepath.Join(abs, "uncompressed_single"+strings.ReplaceAll(ext, ".", "_"))
			must.NoError(t, Compress(abs, src[:1], filepath.Join(abs, src[0]+ext)))
			must.NoError(t, UnCompress(filepath.Join(abs, src[0]+ext), dst))

			data, err := Read(filepath.Join(dst, src[0]))
			check.NoError(t, err)
			check.Equal(t, data, "File 1")
		})
	}
}

func TestListCompress(t *testing.T) {
	abs := t.TempDir()
	src := []string{"list_archive_test1.txt", "list_archive_test2.txt"}
	must.NoError(t, Write(filepath.Join(abs, src[0]), "File 1", 0644))
	must.NoError(t, Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		t.Run(ext, func(t *testing.T) {
			must.NoError(t, Compress(abs, src, filepath.Join(abs, "list_archive_test"+ext)))
			list, err := ListCompress(filepath.Join(abs, "list_archive_test"+ext))
			check.NoError(t, err)
			check.Len(t, list, 2)
		})
	}
	for _, ext := range singleExts {
		t.Run(ext, func(t *testing.T) {
			must.NoError(t, Compress(abs, src[:1], filepath.Join(abs, src[0]+ext)))
			list, err := ListCompress(filepath.Join(abs, src[0]+ext))
			check.NoError(t, err)
			check.DeepEqual(t, list, []string{src[0]})
		})
	}
}

func TestRemoveDeletesFileOrDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remove_test")
	must.NoError(t, os.MkdirAll(path, 0755))
	must.True(t, dirExists(path), must.Msgf("前置目录未建成: %s", path))

	check.NoError(t, Remove(path))
	check.False(t, dirExists(path), check.Msgf("目录应已删除: %s", path))
}

func TestChmodChangesPermissions(t *testing.T) {
	if env.IsWindows() {
		t.Skip("Skipping on Windows")
	}
	path := filepath.Join(t.TempDir(), "chmod_test.txt")
	must.NoError(t, Write(path, "test", 0644))

	check.NoError(t, Chmod(path, 0755))
	info, err := os.Stat(path)
	must.NoError(t, err)
	check.Equal(t, info.Mode().Perm(), os.FileMode(0755))
}

func TestChownChangesOwner(t *testing.T) {
	if env.IsWindows() {
		t.Skip("Skipping on Windows")
	}
	path := filepath.Join(t.TempDir(), "chown_test.txt")
	must.NoError(t, Write(path, "test", 0644))

	// 校验属主是否真的变了需要 root，这里只能确认命令执行成功
	check.NoError(t, Chown(path, "root", "root"))
}

func TestExistsReturnsTrueForExistingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exists_test.txt")
	must.NoError(t, Write(path, "test", 0644))
	check.True(t, Exists(path), check.Msgf("路径应存在: %s", path))
}

func TestExistsReturnsFalseForNonExistingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.txt")
	check.False(t, Exists(path), check.Msgf("路径不应存在: %s", path))
}

func TestEmptyReturnsTrueForEmptyDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty_test")
	must.NoError(t, os.MkdirAll(path, 0755))
	check.True(t, Empty(path), check.Msgf("目录应为空: %s", path))
}

func TestEmptyReturnsFalseForNonEmptyDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonempty_test")
	must.NoError(t, os.MkdirAll(path, 0755))
	must.NoError(t, Write(filepath.Join(path, "file.txt"), "test", 0644))
	check.False(t, Empty(path), check.Msgf("目录不应为空: %s", path))
}

func TestMvMovesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "mv_src.txt")
	dst := filepath.Join(dir, "mv_dst.txt")
	must.NoError(t, Write(src, "test", 0644))

	check.NoError(t, Mv(src, dst))
	check.True(t, fileExists(dst), check.Msgf("目标文件应存在: %s", dst))
	check.False(t, fileExists(src), check.Msgf("源文件应已移走: %s", src))
}

func TestCpCopiesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "cp_src.txt")
	dst := filepath.Join(dir, "cp_dst.txt")
	must.NoError(t, Write(src, "test", 0644))

	check.NoError(t, Cp(src, dst))
	check.True(t, fileExists(dst), check.Msgf("目标文件应存在: %s", dst))
	check.True(t, fileExists(src), check.Msgf("源文件应保留: %s", src))
}

func TestSizeReturnsCorrectSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "size_test.txt")
	data := "12345"
	must.NoError(t, Write(path, data, 0644))

	size, err := Size(path)
	check.NoError(t, err)
	check.Equal(t, size, int64(len(data)))
}

func TestIsDirReturnsTrueForDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "isdir_test")
	must.NoError(t, os.MkdirAll(path, 0755))
	check.True(t, IsDir(path), check.Msgf("应识别为目录: %s", path))
}

func TestIsDirReturnsFalseForFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "isfile_test.txt")
	must.NoError(t, Write(path, "test", 0644))
	check.False(t, IsDir(path), check.Msgf("不应识别为目录: %s", path))
}
