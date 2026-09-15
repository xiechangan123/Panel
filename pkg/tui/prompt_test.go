package tui

import (
	"testing"

	"github.com/libtnb/assert/check"
)

func TestDisplayWidth(t *testing.T) {
	check.Equal(t, displayWidth("abc"), 3)
	check.Equal(t, displayWidth("网站名称"), 8)
	check.Equal(t, displayWidth("? <用户名> › "), 13)
}

func TestFit(t *testing.T) {
	plain := "abcdef"
	check.Equal(t, fit(plain, 10), plain)
	check.Equal(t, fit(plain, 3), "abc\x1b[0m")

	// 颜色序列不计宽，截断处补复位码
	colored := "\x1b[36m网站名称\x1b[0m"
	check.Equal(t, fit(colored, 8), colored)
	check.Equal(t, fit(colored, 5), "\x1b[36m网站\x1b[0m")
}
