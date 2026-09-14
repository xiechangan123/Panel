package systemctl

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/pkg/shell"
)

// ServiceInfo 服务详细信息
type ServiceInfo struct {
	Status string  // 运行状态 (active, inactive, failed, etc.)
	PID    int     // 主进程 PID
	Memory int64   // 内存使用（字节）
	CPU    float64 // CPU 使用率
	Uptime string  // 运行时间
}

// GetServiceInfo 获取服务详细信息
func GetServiceInfo(ctx context.Context, name string) (*ServiceInfo, error) {
	output, err := shell.Execf(ctx, "systemctl show '%s' --property=ActiveState,MainPID,ExecMainStartTimestamp --no-pager", name)
	if err != nil {
		return nil, err
	}

	info := &ServiceInfo{}
	for line := range strings.SplitSeq(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "ActiveState":
			info.Status = value
		case "MainPID":
			info.PID = cast.ToInt(value)
		case "ExecMainStartTimestamp":
			// 格式: Mon 2024-01-01 12:00:00 UTC
			if value != "" && value != "n/a" {
				info.Uptime = calcUptime(value)
			}
		}
	}

	if info.PID > 0 {
		if proc, err := process.NewProcess(int32(info.PID)); err == nil { //nolint:gosec
			// 获取内存信息
			if memInfo, err := proc.MemoryInfo(); err == nil && memInfo != nil {
				info.Memory = int64(memInfo.RSS) //nolint:gosec
			}
			// 获取 CPU 使用率
			if cpu, err := proc.CPUPercent(); err == nil {
				info.CPU = cpu
			}
		}
	}

	return info, nil
}

// Status 获取服务状态
func Status(ctx context.Context, name string) (bool, error) {
	// is-active 在服务未运行时返回退出码 3，只有 ctx 出错才是真失败，否则取消会被误判成「服务未运行」
	output, err := shell.Execf(ctx, "systemctl is-active '%s'", name)
	if err != nil && ctx.Err() != nil {
		return false, ctx.Err()
	}
	return output == "active", nil
}

// IsEnabled 服务是否启用
func IsEnabled(ctx context.Context, name string) (bool, error) {
	// is-enabled 在服务禁用时返回退出码 1，同 Status 只认 ctx 错误
	out, err := shell.Execf(ctx, "systemctl is-enabled '%s'", name)
	if err != nil && ctx.Err() != nil {
		return false, ctx.Err()
	}
	return out == "enabled" || out == "static" || out == "indirect", nil
}

// change 执行 systemd 变更命令
// 变更一旦下发就没有回头路，断开请求 ctx 的取消链，避免连接断开留下「配置改了服务没生效」的半截状态
func change(ctx context.Context, shellCmd string, args ...any) error {
	_, err := shell.ExecfWithTimeout(context.WithoutCancel(ctx), 2*time.Minute, shellCmd, args...)
	return err
}

// Start 启动服务
func Start(ctx context.Context, name string) error {
	return change(ctx, "systemctl start '%s'", name)
}

// Stop 停止服务
func Stop(ctx context.Context, name string) error {
	return change(ctx, "systemctl stop '%s'", name)
}

// Restart 重启服务
func Restart(ctx context.Context, name string) error {
	return change(ctx, "systemctl restart '%s'", name)
}

// Reload 重载服务
func Reload(ctx context.Context, name string) error {
	return change(ctx, "systemctl reload '%s'", name)
}

// Enable 启用服务
func Enable(ctx context.Context, name string) error {
	return change(ctx, "systemctl enable '%s'", name)
}

// Disable 禁用服务
func Disable(ctx context.Context, name string) error {
	return change(ctx, "systemctl disable '%s'", name)
}

// Mask 屏蔽服务
func Mask(ctx context.Context, name string) error {
	return change(ctx, "systemctl mask '%s'", name)
}

// Unmask 解除屏蔽服务
func Unmask(ctx context.Context, name string) error {
	return change(ctx, "systemctl unmask '%s'", name)
}

// Log 获取服务日志
func Log(ctx context.Context, name string) (string, error) {
	return shell.ExecfWithTimeout(ctx, 2*time.Minute, "journalctl -u '%s'", name)
}

// LogTail 获取服务日志
func LogTail(ctx context.Context, name string, lines int) (string, error) {
	return shell.ExecfWithTimeout(ctx, 2*time.Minute, "journalctl -u '%s' --lines '%d'", name, lines)
}

// LogClear 清空服务日志
func LogClear(ctx context.Context, name string) error {
	if _, err := shell.Execf(ctx, "journalctl --rotate -u '%s'", name); err != nil {
		return err
	}
	_, err := shell.Execf(ctx, "journalctl --vacuum-time=1s -u '%s'", name)
	return err
}

// DaemonReload 重载 systemd 服务配置
func DaemonReload(ctx context.Context) error {
	return change(ctx, "systemctl daemon-reload")
}

// calcUptime 计算运行时间
func calcUptime(startTime string) string {
	// 解析时间格式: Mon 2024-01-01 12:00:00 UTC
	// 或者: Mon 2024-01-01 12:00:00 CST
	layouts := []string{
		"Mon 2006-01-02 15:04:05 MST",
		"Mon 2006-01-02 15:04:05 -0700",
	}

	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, startTime)
		if err == nil {
			break
		}
	}
	if err != nil {
		return ""
	}

	duration := time.Since(t)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m"
	}
	if hours > 0 {
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m"
	}
	return strconv.Itoa(minutes) + "m"
}
