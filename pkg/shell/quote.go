package shell

import "strings"

// Quote 单引号包裹拼进 bash 命令的参数
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
