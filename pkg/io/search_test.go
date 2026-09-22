package io

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestSearch(t *testing.T) {
	testDir := filepath.Join(t.TempDir(), "search_test")
	must.NoError(t, os.MkdirAll(filepath.Join(testDir, "subdir"), 0755))
	for path, content := range map[string]string{
		"test_file1.txt":         "内容1",
		"test_file2.log":         "内容2",
		"another_test.txt":       "内容3",
		"subdir/nested_test.txt": "嵌套内容",
		"unrelated.dat":          "无关内容",
	} {
		must.NoError(t, Write(filepath.Join(testDir, path), content, 0644))
	}

	names := func(entries []Entry) []string {
		var names []string
		for _, e := range entries {
			check.Equal(t, filepath.Base(e.Path), e.Info.Name())
			names = append(names, e.Info.Name())
		}
		slices.Sort(names)
		return names
	}

	t.Run("仅当前目录", func(t *testing.T) {
		entries, err := Search(t.Context(), testDir, "test", false)
		must.NoError(t, err)
		// 子目录里的 nested_test.txt 与不匹配的 unrelated.dat 都不该出现
		check.DeepEqual(t, names(entries), []string{"another_test.txt", "test_file1.txt", "test_file2.log"})
	})

	t.Run("递归子目录", func(t *testing.T) {
		entries, err := Search(t.Context(), testDir, "test", true)
		must.NoError(t, err)
		check.DeepEqual(t, names(entries), []string{"another_test.txt", "nested_test.txt", "test_file1.txt", "test_file2.log"})
	})

	t.Run("关键字含通配符按字面匹配", func(t *testing.T) {
		must.NoError(t, Write(filepath.Join(testDir, "photo[1].jpg"), "", 0644))
		entries, err := Search(t.Context(), testDir, "[1]", false)
		must.NoError(t, err)
		check.DeepEqual(t, names(entries), []string{"photo[1].jpg"})
	})

	t.Run("无匹配结果", func(t *testing.T) {
		entries, err := Search(t.Context(), testDir, "nonexistent", false)
		check.NoError(t, err)
		check.Empty(t, entries)
	})

	t.Run("路径不存在", func(t *testing.T) {
		_, err := Search(t.Context(), "/path/does/not/exist", "test", false)
		check.Error(t, err)
	})
}
