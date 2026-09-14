package apache

import (
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
)

// dquote 强制双引号的参数，用于值含特殊编码必须加引号的指令（如 Substitute）
func dquote(value string) conf.Arg {
	return conf.Arg{Value: value, Quote: conf.QuoteDouble}
}

// quote 按引号风格输出参数，自动模式下含空白、引号或尖括号时加双引号
func quote(a conf.Arg) string {
	q := a.Quote
	if q == conf.QuoteAuto {
		q = conf.QuoteNone
		if a.Value == "" || strings.ContainsAny(a.Value, " \t\"<>") {
			q = conf.QuoteDouble
		}
	}
	switch q {
	case conf.QuoteNone:
		return a.Value
	case conf.QuoteSingle:
		return "'" + a.Value + "'"
	default:
		return `"` + strings.ReplaceAll(a.Value, `"`, `\"`) + `"`
	}
}
