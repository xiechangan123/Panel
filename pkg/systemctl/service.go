package systemctl

import (
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"

	"github.com/acepanel/panel/pkg/shell"
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
func GetServiceInfo(name string) (*ServiceInfo, error) {
	output, err := shell.Execf("systemctl show '%s' --property=ActiveState,MainPID,ExecMainStartTimestamp --no-pager", name)
	if err != nil {
		return nil, err
	}

	info := &ServiceInfo{}
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "ActiveState":
			info.Status = value
		case "MainPID":
			if pid, err := strconv.Atoi(value); err == nil {
				info.PID = pid
			}
		case "ExecMainStartTimestamp":
			// 格式: Mon 2024-01-01 12:00:00 UTC
			if value != "" && value != "n/a" {
				info.Uptime = calcUptime(value)
			}
		}
	}

	// 如果有 PID，使用 gopsutil 获取进程信息
	if info.PID > 0 {
		if proc, err := process.NewProcess(int32(info.PID)); err == nil {
			// 获取内存信息
			if memInfo, err := proc.MemoryInfo(); err == nil && memInfo != nil {
				info.Memory = int64(memInfo.RSS)
			}
			// 获取 CPU 使用率
			if cpu, err := proc.CPUPercent(); err == nil {
				info.CPU = cpu
			}
		}
	}

	return info, nil
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

// Status 获取服务状态
func Status(name string) (bool, error) {
	output, _ := shell.Execf("systemctl is-active '%s'", name) // 不判断错误，因为 is-active 在服务未启用时会返回 3
	return output == "active", nil
}

// IsEnabled 服务是否启用
func IsEnabled(name string) (bool, error) {
	out, _ := shell.Execf("systemctl is-enabled '%s'", name) // 不判断错误，因为 is-enabled 在服务禁用时会返回 1
	return out == "enabled" || out == "static" || out == "indirect", nil
}

// Start 启动服务
func Start(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl start '%s'", name)
	return err
}

// Stop 停止服务
func Stop(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl stop '%s'", name)
	return err
}

// Restart 重启服务
func Restart(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl restart '%s'", name)
	return err
}

// Reload 重载服务
func Reload(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl reload '%s'", name)
	return err
}

// Enable 启用服务
func Enable(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl enable '%s'", name)
	return err
}

// Disable 禁用服务
func Disable(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl disable '%s'", name)
	return err
}

// Mask 屏蔽服务
func Mask(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl mask '%s'", name)
	return err
}

// Unmask 解除屏蔽服务
func Unmask(name string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl unmask '%s'", name)
	return err
}

// Log 获取服务日志
func Log(name string) (string, error) {
	return shell.ExecfWithTimeout(2*time.Minute, "journalctl -u '%s'", name)
}

// LogTail 获取服务日志
func LogTail(name string, lines int) (string, error) {
	return shell.ExecfWithTimeout(2*time.Minute, "journalctl -u '%s' --lines '%d'", name, lines)
}

// LogClear 清空服务日志
func LogClear(name string) error {
	if _, err := shell.Execf("journalctl --rotate -u '%s'", name); err != nil {
		return err
	}
	_, err := shell.Execf("journalctl --vacuum-time=1s -u '%s'", name)
	return err
}

// DaemonReload 重载 systemd 服务配置
func DaemonReload() error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "systemctl daemon-reload")
	return err
}
