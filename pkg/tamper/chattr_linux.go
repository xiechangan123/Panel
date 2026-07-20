//go:build linux

package tamper

import (
	"os"

	"github.com/acepanel/panel/v3/pkg/chattr"
)

// chattrEngine 基于 inode 属性锁定的防篡改引擎
// 对文件加 immutable(+i),对目录加 append-only(+a)
type chattrEngine struct{}

func newChattrEngine() *chattrEngine {
	return &chattrEngine{}
}

// setFlag 对单个路径设置属性
func setFlag(path string, flag uint32) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return chattr.SetAttr(f, flag)
}

// unsetFlag 对单个路径解除属性
func unsetFlag(path string, flag uint32) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return chattr.UnsetAttr(f, flag)
}

// apply 先锁文件(+i)再锁目录(+a),目录锁定后其下条目不可增删
func (c *chattrEngine) apply(entries []fileEntry) error {
	for _, e := range entries {
		if e.isDir {
			continue
		}
		_ = setFlag(e.path, chattr.FS_IMMUTABLE_FL)
	}
	for _, e := range entries {
		if !e.isDir {
			continue
		}
		_ = setFlag(e.path, chattr.FS_APPEND_FL)
	}
	return nil
}

// remove 先解目录再解文件,与 apply 逆序
func (c *chattrEngine) remove(entries []fileEntry) error {
	for _, e := range entries {
		if !e.isDir {
			continue
		}
		_ = unsetFlag(e.path, chattr.FS_APPEND_FL)
	}
	for _, e := range entries {
		if e.isDir {
			continue
		}
		_ = unsetFlag(e.path, chattr.FS_IMMUTABLE_FL)
	}
	return nil
}

// lockOne 锁定单个新增文件
func (c *chattrEngine) lockOne(path string, isDir bool) {
	if isDir {
		_ = setFlag(path, chattr.FS_APPEND_FL)
	} else {
		_ = setFlag(path, chattr.FS_IMMUTABLE_FL)
	}
}

// events chattr 模式无内核事件,拦截靠属性,监控靠 fsnotify
func (c *chattrEngine) events() <-chan Event {
	return nil
}

func (c *chattrEngine) close() error {
	return nil
}
