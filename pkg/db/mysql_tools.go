package db

import (
	"fmt"
	"os"
	"regexp"

	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

// MySQLResetRootPassword 重置 MySQL root密码
func MySQLResetRootPassword(password string) error {
	_ = systemctl.Stop("mysqld")
	if run, err := systemctl.Status("mysqld"); err != nil || run {
		return fmt.Errorf("failed to stop MySQL: %w", err)
	}
	_, _ = shell.Execf(`systemctl set-environment MYSQLD_OPTS="--skip-grant-tables --skip-networking"`)
	if err := systemctl.Start("mysqld"); err != nil {
		return fmt.Errorf("failed to start MySQL in safe mode: %w", err)
	}
	if _, err := shell.Execf(`mysql -uroot -e "FLUSH PRIVILEGES;UPDATE mysql.user SET authentication_string=null WHERE user='root' AND host='localhost';ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';FLUSH PRIVILEGES;"`, password); err != nil {
		return fmt.Errorf("failed to reset MySQL root password: %w", err)
	}
	if err := systemctl.Stop("mysqld"); err != nil {
		return fmt.Errorf("failed to stop MySQL: %w", err)
	}
	_, _ = shell.Execf(`systemctl unset-environment MYSQLD_OPTS`)
	if err := systemctl.Start("mysqld"); err != nil {
		return fmt.Errorf("failed to start MySQL: %w", err)
	}

	return nil
}

// MySQLSocket 探测本地 MySQL 的 unix socket 路径
// 依次检查 /tmp/mysql.sock 及传入配置文件中的 socket 配置，均未命中返回空
func MySQLSocket(configs ...string) string {
	if _, err := os.Stat("/tmp/mysql.sock"); err == nil {
		return "/tmp/mysql.sock"
	}
	re := regexp.MustCompile(`socket\s*=\s*['"]?([^'"\s]+)`)
	for _, conf := range configs {
		content, err := os.ReadFile(conf)
		if err != nil {
			continue
		}
		if matches := re.FindStringSubmatch(string(content)); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}
