package db

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// PostgresPort 读取本地 PostgreSQL 端口
func PostgresPort(root string) uint {
	content, err := os.ReadFile(filepath.Join(root, "server/postgresql/data/postgresql.conf"))
	if err != nil {
		return 5432
	}
	// 注释行以 # 开头，不会被 ^\s*port 命中
	re := regexp.MustCompile(`(?m)^\s*port\s*=\s*'?(\d+)`)
	if matches := re.FindStringSubmatch(string(content)); len(matches) > 1 {
		if port, err := strconv.ParseUint(matches[1], 10, 32); err == nil && port != 0 {
			return uint(port)
		}
	}
	return 5432
}
