package tamper

import "time"

// Mode 防篡改模式
type Mode string

const (
	// ModeChattr 文件属性锁定(chattr +i/+a),通用、事前内核强制、不防 root
	ModeChattr Mode = "chattr"
	// ModeEBPF eBPF-LSM 拦截,精准可溯源,需内核激活 bpf LSM
	ModeEBPF Mode = "ebpf"
)

// Op 被拦截的操作类型
type Op uint32

const (
	OpWrite   Op = iota // 写打开
	OpUnlink            // 删除
	OpRename            // 重命名
	OpSetattr           // 属性修改(chmod/chown/truncate)
	OpCreate            // 新建(用户态监控)
)

func (o Op) String() string {
	switch o {
	case OpWrite:
		return "write"
	case OpUnlink:
		return "unlink"
	case OpRename:
		return "rename"
	case OpSetattr:
		return "setattr"
	case OpCreate:
		return "create"
	default:
		return "unknown"
	}
}

// Event 一次篡改拦截/告警事件
type Event struct {
	Path  string    `json:"path"`
	Dev   uint64    `json:"-"` // 设备号,与 inode 共同唯一标识文件(内部使用)
	Inode uint64    `json:"inode"`
	PID   uint32    `json:"pid"`
	Comm  string    `json:"comm"`
	Op    Op        `json:"op"`
	OpStr string    `json:"op_str"`
	Time  time.Time `json:"time"`
}

// Rule 单个保护目标的规则(通常对应一个网站)
type Rule struct {
	Name     string   `json:"name"`     // 标识(网站名)
	Paths    []string `json:"paths"`    // 受保护目录
	Exts     []string `json:"exts"`     // 受保护后缀(不含点,空表示全部文件)
	Excludes []string `json:"excludes"` // 排除的子路径(相对或绝对)
}

// Config 防篡改运行配置
type Config struct {
	Mode          Mode   // 保护模式
	Rules         []Rule // 保护规则集
	BlockNewFiles bool   // 受保护目录下新建受保护类型文件时删除拦截(仅记录 vs 删除)
}

// Stats 运行统计
type Stats struct {
	Mode           Mode `json:"mode"`
	Running        bool `json:"running"`
	ProtectedFiles int  `json:"protected_files"`
	ProtectedDirs  int  `json:"protected_dirs"`
}

// EBPFStatus eBPF 可用性检测结果
type EBPFStatus struct {
	Available     bool   `json:"available"`      // 当前是否可直接使用
	KernelVersion string `json:"kernel_version"` // 内核版本
	BPFLSMActive  bool   `json:"bpf_lsm_active"` // bpf 是否在激活的 LSM 列表中
	ActiveLSM     string `json:"active_lsm"`     // 当前激活的 LSM 列表
	Reason        string `json:"reason"`         // 不可用原因
}
