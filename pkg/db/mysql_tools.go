package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

// MySQLResetRootPassword 重置 MySQL root密码
func MySQLResetRootPassword(password, root string) (err error) {
	_ = systemctl.Stop("mysqld")
	if run, _ := systemctl.Status("mysqld"); run {
		return errors.New("failed to stop MySQL")
	}

	if _, err = shell.Execf(`systemctl set-environment MYSQLD_OPTS="--skip-grant-tables --skip-networking"`); err != nil {
		return fmt.Errorf("failed to enter MySQL safe mode: %w", err)
	}
	defer func() {
		_, _ = shell.Execf(`systemctl unset-environment MYSQLD_OPTS`)
		if rerr := systemctl.Restart("mysqld"); rerr != nil && err == nil {
			err = fmt.Errorf("failed to restart MySQL: %w", rerr)
		}
	}()

	if err = systemctl.Start("mysqld"); err != nil {
		return fmt.Errorf("failed to start MySQL in safe mode: %w", err)
	}

	// 此刻实例刚以安全模式起来，socket 已按配置重建
	socket := ""
	if sock := MySQLSocket(root); sock != "" {
		socket = "--socket=" + sock
	}
	// FLUSH PRIVILEGES 让跳过校验启动的实例重新加载权限表，之后 ALTER USER 才可用
	if _, err = shell.Execf(
		`mysql -uroot %s -e "FLUSH PRIVILEGES;ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';FLUSH PRIVILEGES;"`,
		socket,
		password,
	); err != nil {
		return fmt.Errorf("failed to reset MySQL root password: %w", err)
	}

	return nil
}

// MySQLSocket 探测本地 MySQL 的 unix socket 路径
func MySQLSocket(root string) string {
	if _, err := os.Stat("/tmp/mysql.sock"); err == nil {
		return "/tmp/mysql.sock"
	}
	re := regexp.MustCompile(`socket\s*=\s*['"]?([^'"\s]+)`)
	for _, conf := range []string{filepath.Join(root, "server/mysql/config/my.cnf"), "/etc/my.cnf"} {
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
