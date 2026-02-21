//go:build !unix

package ipdb

import "os"

type mmapFile struct {
	data []byte
}

// mmapOpen 读取文件到内存，非 Unix 平台回退方案
func mmapOpen(path string) (*mmapFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrInvalidFile
	}
	return &mmapFile{data: data}, nil
}

// Close 释放内存引用
func (m *mmapFile) Close() error {
	m.data = nil
	return nil
}
