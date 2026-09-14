package io

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestSearchX(t *testing.T) {
	testDir := filepath.Join(t.TempDir(), "search_test")
	must.NoError(t, os.MkdirAll(testDir, 0755))
	must.NoError(t, os.MkdirAll(filepath.Join(testDir, "subdir"), 0755))

	testFiles := map[string]string{
		"test_file1.txt":         "内容1",
		"test_file2.log":         "内容2",
		"another_test.txt":       "内容3",
		"subdir/nested_test.txt": "嵌套内容",
		"unrelated.dat":          "无关内容",
	}

	for path, content := range testFiles {
		must.NoError(t, Write(filepath.Join(testDir, path), content, 0644))
	}

	t.Run("正常搜索", func(t *testing.T) {
		entries, err := SearchX(t.Context(), testDir, "test", false)
		must.NoError(t, err)

		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
			info, err := entry.Info()
			must.NoError(t, err)
			check.Equal(t, info.Mode().Type(), entry.Type())
			check.Equal(t, info.IsDir(), entry.IsDir())
		}
		slices.Sort(names)

		// 子目录里的 nested_test.txt 与不匹配的 unrelated.dat 都不该出现
		check.DeepEqual(t, names, []string{"another_test.txt", "test_file1.txt", "test_file2.log"})
	})

	t.Run("无匹配结果", func(t *testing.T) {
		entries, err := SearchX(t.Context(), testDir, "nonexistent", false)
		check.NoError(t, err)
		check.Empty(t, entries)
	})

	t.Run("路径不存在", func(t *testing.T) {
		_, err := SearchX(t.Context(), "/path/does/not/exist", "test", false)
		check.Error(t, err)
	})
}
