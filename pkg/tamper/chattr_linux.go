//go:build linux

package tamper

import (
	"os"

	"github.com/acepanel/panel/v3/pkg/chattr"
)

// chattrEngine 文件 +i(immutable),整树目录 +a(append-only);扩展名目录不锁以允许无关文件写入
type chattrEngine struct{}

func newChattrEngine() *chattrEngine { return &chattrEngine{} }

func setFlag(path string, flag uint32) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return chattr.SetAttr(f, flag)
}

func unsetFlag(path string, flag uint32) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return chattr.UnsetAttr(f, flag)
}

func (c *chattrEngine) apply(entries []fileEntry) error {
	for _, e := range entries {
		if e.isDir {
			continue
		}
		_ = setFlag(e.path, chattr.FS_IMMUTABLE_FL)
	}
	for _, e := range entries {
		if !e.isDir || len(e.exts) > 0 {
			continue
		}
		_ = setFlag(e.path, chattr.FS_APPEND_FL)
	}
	return nil
}

// remove 与 apply 逆序,先解目录再解文件
func (c *chattrEngine) remove(entries []fileEntry) error {
	for _, e := range entries {
		if !e.isDir || len(e.exts) > 0 {
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

func (c *chattrEngine) lockOne(path string, isDir bool) {
	if isDir {
		_ = setFlag(path, chattr.FS_APPEND_FL)
	} else {
		_ = setFlag(path, chattr.FS_IMMUTABLE_FL)
	}
}

func (c *chattrEngine) start() error         { return nil }
func (c *chattrEngine) events() <-chan Event { return nil }
func (c *chattrEngine) close() error         { return nil }
