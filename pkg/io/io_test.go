package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libtnb/utils/env"
	"github.com/stretchr/testify/suite"
)

type IOTestSuite struct {
	suite.Suite
}

func TestIOTestSuite(t *testing.T) {
	suite.Run(t, &IOTestSuite{})
}

func (s *IOTestSuite) SetupTest() {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		s.NoError(os.MkdirAll("testdata", 0755))
	}
}

func (s *IOTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll("testdata"))
}

func (s *IOTestSuite) TestWriteCreatesFileWithCorrectContent() {
	path := "testdata/write_test.txt"
	data := "Hello, World!"
	permission := os.FileMode(0644)

	s.NoError(Write(path, data, permission))

	content, err := Read(path)
	s.NoError(err)
	s.Equal(data, content)
}

func (s *IOTestSuite) TestWriteAppendAppendsToFile() {
	path := "testdata/append_test.txt"
	initialData := "Hello"
	appendData := ", World!"

	s.NoError(Write(path, initialData, 0644))
	s.NoError(WriteAppend(path, appendData, 0644))

	content, err := Read(path)
	s.NoError(err)
	s.Equal("Hello, World!", content)
}

// archiveExts 归档格式，支持多文件压缩
var archiveExts = []string{".zip", ".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tar.xz", ".tar.zst", ".7z"}

// singleExts 单文件压缩格式，仅支持压缩单个文件
var singleExts = []string{".gz", ".bz2", ".xz", ".zst"}

func (s *IOTestSuite) TestCompress() {
	abs, err := filepath.Abs("testdata")
	s.NoError(err)
	src := []string{"compress_test1.txt", "compress_test2.txt"}
	s.NoError(Write(filepath.Join(abs, src[0]), "File 1", 0644))
	s.NoError(Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		s.NoError(Compress(abs, src, filepath.Join(abs, "compress_test"+ext)), ext)
	}
	for _, ext := range singleExts {
		s.NoError(Compress(abs, src[:1], filepath.Join(abs, "compress_single"+ext)), ext)
		s.Error(Compress(abs, src, filepath.Join(abs, "compress_multi"+ext)), ext)
	}
}

func (s *IOTestSuite) TestUnCompress() {
	abs, err := filepath.Abs("testdata")
	s.NoError(err)
	src := []string{"uncompress_test1.txt", "uncompress_test2.txt"}
	s.NoError(Write(filepath.Join(abs, src[0]), "File 1", 0644))
	s.NoError(Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		dst := filepath.Join(abs, "uncompressed"+strings.ReplaceAll(ext, ".", "_"))
		s.NoError(Compress(abs, src, filepath.Join(abs, "uncompress_test"+ext)), ext)
		s.NoError(UnCompress(filepath.Join(abs, "uncompress_test"+ext), dst), ext)
		data, err := Read(filepath.Join(dst, src[0]))
		s.NoError(err, ext)
		s.Equal("File 1", data, ext)
		data, err = Read(filepath.Join(dst, src[1]))
		s.NoError(err, ext)
		s.Equal("File 2", data, ext)
	}
	// 单文件压缩格式解压后去掉压缩后缀恢复原文件名
	for _, ext := range singleExts {
		dst := filepath.Join(abs, "uncompressed_single"+strings.ReplaceAll(ext, ".", "_"))
		s.NoError(Compress(abs, src[:1], filepath.Join(abs, src[0]+ext)), ext)
		s.NoError(UnCompress(filepath.Join(abs, src[0]+ext), dst), ext)
		data, err := Read(filepath.Join(dst, src[0]))
		s.NoError(err, ext)
		s.Equal("File 1", data, ext)
	}
}

func (s *IOTestSuite) TestListCompress() {
	abs, err := filepath.Abs("testdata")
	s.NoError(err)
	src := []string{"list_archive_test1.txt", "list_archive_test2.txt"}
	s.NoError(Write(filepath.Join(abs, src[0]), "File 1", 0644))
	s.NoError(Write(filepath.Join(abs, src[1]), "File 2", 0644))

	for _, ext := range archiveExts {
		s.NoError(Compress(abs, src, filepath.Join(abs, "list_archive_test"+ext)), ext)
		list, err := ListCompress(filepath.Join(abs, "list_archive_test"+ext))
		s.NoError(err, ext)
		s.Len(list, 2, ext)
	}
	// 单文件压缩格式返回去掉压缩后缀的文件名
	for _, ext := range singleExts {
		s.NoError(Compress(abs, src[:1], filepath.Join(abs, src[0]+ext)), ext)
		list, err := ListCompress(filepath.Join(abs, src[0]+ext))
		s.NoError(err, ext)
		s.Equal([]string{src[0]}, list, ext)
	}
}

func (s *IOTestSuite) TestRemoveDeletesFileOrDirectory() {
	path := "testdata/remove_test"
	s.NoError(os.MkdirAll(path, 0755))
	s.DirExists(path)

	s.NoError(Remove(path))
	s.NoDirExists(path)
}

func (s *IOTestSuite) TestChmodChangesPermissions() {
	if env.IsWindows() {
		s.T().Skip("Skipping on Windows")
	}
	path := "testdata/chmod_test.txt"
	s.NoError(Write(path, "test", 0644))

	s.NoError(Chmod(path, 0755))
	info, err := os.Stat(path)
	s.NoError(err)
	s.Equal(os.FileMode(0755), info.Mode().Perm())
}

func (s *IOTestSuite) TestChownChangesOwner() {
	if env.IsWindows() {
		s.T().Skip("Skipping on Windows")
	}
	path := "testdata/chown_test.txt"
	s.NoError(Write(path, "test", 0644))

	s.NoError(Chown(path, "root", "root"))
}

func (s *IOTestSuite) TestExistsReturnsTrueForExistingPath() {
	path := "testdata/exists_test.txt"
	s.NoError(Write(path, "test", 0644))
	s.True(Exists(path))
}

func (s *IOTestSuite) TestExistsReturnsFalseForNonExistingPath() {
	path := "testdata/nonexistent.txt"
	s.False(Exists(path))
}

func (s *IOTestSuite) TestEmptyReturnsTrueForEmptyDirectory() {
	path := "testdata/empty_test"
	s.NoError(os.MkdirAll(path, 0755))
	s.True(Empty(path))
}

func (s *IOTestSuite) TestEmptyReturnsFalseForNonEmptyDirectory() {
	path := "testdata/nonempty_test"
	s.NoError(os.MkdirAll(path, 0755))
	s.NoError(Write(filepath.Join(path, "file.txt"), "test", 0644))
	s.False(Empty(path))
}

func (s *IOTestSuite) TestMvMovesFile() {
	src := "testdata/mv_src.txt"
	dst := "testdata/mv_dst.txt"
	s.NoError(Write(src, "test", 0644))

	s.NoError(Mv(src, dst))
	s.FileExists(dst)
	s.NoFileExists(src)
}

func (s *IOTestSuite) TestCpCopiesFile() {
	src := "testdata/cp_src.txt"
	dst := "testdata/cp_dst.txt"
	s.NoError(Write(src, "test", 0644))

	s.NoError(Cp(src, dst))
	s.FileExists(dst)
	s.FileExists(src)
}

func (s *IOTestSuite) TestSizeReturnsCorrectSize() {
	path := "testdata/size_test.txt"
	data := "12345"
	s.NoError(Write(path, data, 0644))

	size, err := Size(path)
	s.NoError(err)
	s.Equal(int64(len(data)), size)
}

func (s *IOTestSuite) TestIsDirReturnsTrueForDirectory() {
	path := "testdata/isdir_test"
	s.NoError(os.MkdirAll(path, 0755))
	s.True(IsDir(path))
}

func (s *IOTestSuite) TestIsDirReturnsFalseForFile() {
	path := "testdata/isfile_test.txt"
	s.NoError(Write(path, "test", 0644))
	s.False(IsDir(path))
}
