package apache

import (
	"os"
)

// ParseOptions 定义解析器选项
type ParseOptions struct {
	// ProcessIncludes 是否处理Include指令，递归加载包含的文件
	ProcessIncludes bool

	// BaseDir 基础目录，用于解析相对路径的Include文件
	BaseDir string

	// MaxIncludeDepth 最大包含深度，防止无限递归
	MaxIncludeDepth int
}

// DefaultParseOptions 返回默认的解析选项
func DefaultParseOptions() *ParseOptions {
	wd, _ := os.Getwd()
	return &ParseOptions{
		ProcessIncludes: false, // 默认不处理Include
		BaseDir:         wd,
		MaxIncludeDepth: 10,
	}
}
