package tamper

import "time"

type Mode string

const (
	ModeChattr Mode = "chattr" // 通用,不防 root
	ModeEBPF   Mode = "ebpf"   // 需内核激活 bpf LSM
)

type Op uint32

const (
	OpWrite Op = iota
	OpUnlink
	OpRename
	OpSetattr
	OpCreate
	OpLink
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
	case OpLink:
		return "link"
	default:
		return "unknown"
	}
}

type Event struct {
	Path   string    `json:"path"`
	Dev    uint64    `json:"-"`
	Name   string    `json:"-"` // 创建类事件的新条目名
	Denied bool      `json:"-"` // 内核已拒绝;创建类事件区分拦截/放行观察,存量拦截恒真
	Inode  uint64    `json:"inode"`
	PID    uint32    `json:"pid"`
	Comm   string    `json:"comm"`
	Op     Op        `json:"op"`
	OpStr  string    `json:"op_str"`
	Time   time.Time `json:"time"`
}

type Rule struct {
	Name     string   `json:"name"`
	Paths    []string `json:"paths"`
	Exts     []string `json:"exts"`     // 受保护后缀(不含点),空=全部文件
	Excludes []string `json:"excludes"` // 排除的子路径(相对或绝对)
}

type Config struct {
	Mode          Mode
	Rules         []Rule
	BlockNewFiles bool // 关闭时仅记录不拦截(eBPF 内核拒绝/chattr 删除)
}

type Stats struct {
	Mode           Mode `json:"mode"`
	Running        bool `json:"running"`
	ProtectedFiles int  `json:"protected_files"`
	ProtectedDirs  int  `json:"protected_dirs"`
}

type EBPFStatus struct {
	Available     bool   `json:"available"`
	KernelVersion string `json:"kernel_version"`
	BPFLSMActive  bool   `json:"bpf_lsm_active"`
	ActiveLSM     string `json:"active_lsm"`
	Reason        string `json:"reason"`
}
