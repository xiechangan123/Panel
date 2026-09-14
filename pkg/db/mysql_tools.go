package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

// MySQLResetRootPassword 重置 MySQL root密码
func MySQLResetRootPassword(ctx context.Context, password, root string) (err error) {
	_ = systemctl.Stop(ctx, "mysqld")
	// 状态查不出来时不能当作「已停止」继续，否则会带着一个还活着的实例进安全模式
	if run, serr := systemctl.Status(ctx, "mysqld"); serr != nil || run {
		return errors.New("failed to stop MySQL")
	}

	// 实例此刻是停的，收尾装在改环境变量之前，任意一步失败都能把实例拉回来
	// 收尾不能跟随 ctx 取消，否则调用方取消时实例会停在安全模式
	cleanupCtx := context.WithoutCancel(ctx)
	defer func() {
		_, _ = shell.Execf(cleanupCtx, `systemctl unset-environment MYSQLD_OPTS`)
		if rerr := systemctl.Restart(cleanupCtx, "mysqld"); rerr != nil && err == nil {
			err = fmt.Errorf("failed to restart MySQL: %w", rerr)
		}
	}()

	if _, err = shell.Execf(ctx, `systemctl set-environment MYSQLD_OPTS="--skip-grant-tables --skip-networking"`); err != nil {
		return fmt.Errorf("failed to enter MySQL safe mode: %w", err)
	}

	if err = systemctl.Start(ctx, "mysqld"); err != nil {
		return fmt.Errorf("failed to start MySQL in safe mode: %w", err)
	}

	// 此刻实例刚以安全模式起来，socket 已按配置重建
	socket := ""
	if sock := MySQLSocket(root); sock != "" {
		socket = "--socket=" + sock
	}
	// FLUSH PRIVILEGES 让跳过校验启动的实例重新加载权限表，之后 ALTER USER 才可用
	if _, err = shell.Execf(
		ctx, `mysql -uroot %s -e "FLUSH PRIVILEGES;ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';FLUSH PRIVILEGES;"`,
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
