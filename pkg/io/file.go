package io

import (
	"os"
	"path/filepath"
)

// Write 目标带 +i/+a 属性（如 .user.ini）时先解锁，写完恢复
func Write(path string, data string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if lf, ok := unlockEntry(path); ok {
		defer relockAttr([]lockedFile{lf})
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = file.WriteString(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Read 读取文件
func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}
