package types

import (
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// CurrentInfo 监控信息
type CurrentInfo struct {
	Cpus         []cpu.InfoStat         `json:"cpus"`
	Percent      float64                `json:"percent"`  // 总使用率
	Percents     []float64              `json:"percents"` // 每个核心使用率
	Load         *load.AvgStat          `json:"load"`
	Host         *host.InfoStat         `json:"host"`
	Mem          *mem.VirtualMemoryStat `json:"mem"`
	Swap         *mem.SwapMemoryStat    `json:"swap"`
	Net          []net.IOCountersStat   `json:"net"`
	DiskIO       []disk.IOCountersStat  `json:"disk_io"`
	Disk         []disk.PartitionStat   `json:"disk"`
	DiskUsage    []disk.UsageStat       `json:"disk_usage"`
	Time         time.Time              `json:"time"`
	TopProcesses TopProcesses           `json:"top_processes"`
}

// ProcessStat 进程统计快照
type ProcessStat struct {
	PID      int32   `json:"pid"`
	Name     string  `json:"name"`
	Username string  `json:"username"`
	Command  string  `json:"command"`
	Value    float64 `json:"value"`           // 主指标值：CPU%、内存MB、IO字节
	Read     float64 `json:"read,omitempty"`  // 仅磁盘IO：读取字节
	Write    float64 `json:"write,omitempty"` // 仅磁盘IO：写入字节
}

// TopProcesses 各指标 Top 5 进程
type TopProcesses struct {
	CPU    []ProcessStat `json:"cpu"`
	Memory []ProcessStat `json:"memory"`
	DiskIO []ProcessStat `json:"disk_io"`
}
