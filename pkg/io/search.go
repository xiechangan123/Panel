package io

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/acepanel/panel/pkg/shell"
)

// Search 查找文件/文件夹
func Search(path, keyword string, sub bool) (map[string]os.FileInfo, error) {
	paths := make(map[string]os.FileInfo)
	baseDepth := strings.Count(filepath.Clean(path), string(os.PathSeparator))

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !sub && strings.Count(p, string(os.PathSeparator)) > baseDepth+1 {
			return filepath.SkipDir
		}
		if strings.Contains(info.Name(), keyword) {
			paths[p] = info
		}
		return nil
	})

	return paths, err
}

// SearchX 查找文件/文件夹（find命令）
func SearchX(path, keyword string, sub bool) ([]os.DirEntry, error) {
	var out string
	var err error
	if sub {
		out, err = shell.Execf("find '%s' -name '*%s*'", path, keyword)
	} else {
		out, err = shell.Execf("find '%s' -maxdepth 1 -name '*%s*'", path, keyword)
	}
	if err != nil {
		return nil, err
	}

	var entries []os.DirEntry
	lines := strings.SplitSeq(out, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == path {
			continue
		}
		entry, err := newSearchEntryFromPath(line)
		if err != nil {
			continue // 直接跳过，不返回错误，不然很烦人的
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// SearchEntry 实现 os.DirEntry 接口
type SearchEntry struct {
	path string
	info os.FileInfo
}

// newSearchEntryFromPath 根据文件路径创建 SearchEntry
func newSearchEntryFromPath(path string) (*SearchEntry, error) {
	info, err := os.Lstat(path) // 不跟随符号链接
	if err != nil {
		return nil, err
	}
	return &SearchEntry{path: path, info: info}, nil
}

// Name 返回文件基本名称
func (d *SearchEntry) Name() string {
	return filepath.Base(d.path)
}

// IsDir 判断是否为目录
func (d *SearchEntry) IsDir() bool {
	return d.info.IsDir()
}

// Type 返回文件模式类型
func (d *SearchEntry) Type() os.FileMode {
	return d.info.Mode().Type()
}

// Info 返回文件信息
func (d *SearchEntry) Info() (os.FileInfo, error) {
	return d.info, nil
}

// Path 返回文件完整路径
func (d *SearchEntry) Path() string {
	return d.path
}
