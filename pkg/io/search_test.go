package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SearchTestSuite struct {
	suite.Suite
}

func TestSearchTestSuite(t *testing.T) {
	suite.Run(t, &SearchTestSuite{})
}

func (s *SearchTestSuite) SetupTest() {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		s.NoError(os.MkdirAll("testdata", 0755))
	}
}

func (s *SearchTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll("testdata"))
}

func (s *SearchTestSuite) TestSearchX() {
	testDir := "testdata/search_test"
	s.NoError(os.MkdirAll(testDir, 0755))
	s.NoError(os.MkdirAll(filepath.Join(testDir, "subdir"), 0755))

	testFiles := map[string]string{
		"test_file1.txt":         "内容1",
		"test_file2.log":         "内容2",
		"another_test.txt":       "内容3",
		"subdir/nested_test.txt": "嵌套内容",
		"unrelated.dat":          "无关内容",
	}

	for path, content := range testFiles {
		s.NoError(Write(filepath.Join(testDir, path), content, 0644))
	}

	s.Run("正常搜索", func() {
		entries, err := SearchX(testDir, "test", false)
		s.NoError(err)

		names := make(map[string]bool)
		for _, entry := range entries {
			names[entry.Name()] = true
			s.NotEmpty(entry.Name())
			info, err := entry.Info()
			s.NoError(err)
			s.NotNil(info)
			s.Equal(entry.Type(), info.Mode().Type())
			s.Equal(entry.IsDir(), info.IsDir())
		}

		s.True(names["test_file1.txt"])
		s.True(names["test_file2.log"])
		s.True(names["another_test.txt"])
		s.False(names["nested_test.txt"]) // 不应该找到子目录中的文件
		s.False(names["unrelated.dat"])   // 不应该找到不匹配的文件
	})

	s.Run("无匹配结果", func() {
		entries, err := SearchX(testDir, "nonexistent", false)
		s.NoError(err)
		s.Empty(entries)
	})

	s.Run("路径不存在", func() {
		_, err := SearchX("/path/does/not/exist", "test", false)
		s.Error(err)
	})
}
