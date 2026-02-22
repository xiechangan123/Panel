package ipdb

import (
	"os"
	"syscall"
)

type mmapFile struct {
	data []byte
}

// mmapOpen 通过 mmap 映射文件到内存
func mmapOpen(path string) (*mmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := int(info.Size())
	if size == 0 {
		return nil, ErrInvalidFile
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	return &mmapFile{data: data}, nil
}

// Close 释放 mmap 映射
func (m *mmapFile) Close() error {
	if m.data == nil {
		return nil
	}
	err := syscall.Munmap(m.data)
	m.data = nil
	return err
}
