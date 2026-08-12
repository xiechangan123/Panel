package types

import (
	"net/netip"
	"time"
)

// MigrationStep 迁移步骤
type MigrationStep string

const (
	MigrationStepIdle     MigrationStep = "idle"     // 空闲
	MigrationStepConnect  MigrationStep = "connect"  // 连接信息
	MigrationStepPreCheck MigrationStep = "precheck" // 预检查
	MigrationStepSelect   MigrationStep = "select"   // 选择迁移项
	MigrationStepRunning  MigrationStep = "running"  // 迁移中
	MigrationStepDone     MigrationStep = "done"     // 迁移完成
)

// MigrationItemStatus 迁移项状态
type MigrationItemStatus string

const (
	MigrationItemPending MigrationItemStatus = "pending"
	MigrationItemRunning MigrationItemStatus = "running"
	MigrationItemSuccess MigrationItemStatus = "success"
	MigrationItemPartial MigrationItemStatus = "partial"
	MigrationItemFailed  MigrationItemStatus = "failed"
	MigrationItemSkipped MigrationItemStatus = "skipped"
)

// MigrationItemStage 迁移项当前阶段
type MigrationItemStage string

const (
	MigrationStagePreparing   MigrationItemStage = "preparing"
	MigrationStageDownloading MigrationItemStage = "downloading"
	MigrationStageValidating  MigrationItemStage = "validating"
	MigrationStageImporting   MigrationItemStage = "importing"
	MigrationStageConfiguring MigrationItemStage = "configuring"
	MigrationStageStarting    MigrationItemStage = "starting"
	MigrationStageDone        MigrationItemStage = "done"
)

// MigrationItemResult 单个迁移项的结果
type MigrationItemResult struct {
	Key       string              `json:"key"`
	Type      string              `json:"type"`   // website / database / project
	Name      string              `json:"name"`   // 名称
	Status    MigrationItemStatus `json:"status"` // 状态
	Stage     MigrationItemStage  `json:"stage"`
	Error     string              `json:"error"` // 失败原因
	Warnings  []string            `json:"warnings"`
	Created   []string            `json:"created_resources"`
	Residuals []string            `json:"residual_resources"`
	StartedAt *time.Time          `json:"started_at"` // 开始时间
	EndedAt   *time.Time          `json:"ended_at"`   // 结束时间
	Duration  float64             `json:"duration"`   // 耗时（秒）
}

// MigrationSourceInfo 来源面板及服务器信息
type MigrationSourceInfo struct {
	Panel        string   `json:"panel"`
	Version      string   `json:"version"`
	Architecture string   `json:"architecture"`
	Hostname     string   `json:"hostname"`
	Capabilities []string `json:"capabilities"`
	Warnings     []string `json:"warnings"`
}

// MigrationSourceItem 来源面板上的统一迁移资源
type MigrationSourceItem struct {
	Key            string   `json:"key"`
	Type           string   `json:"type"`
	Subtype        string   `json:"subtype"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	Size           int64    `json:"size"`
	Supported      bool     `json:"supported"`
	Blockers       []string `json:"blockers"`
	Warnings       []string `json:"warnings"`
	Features       []string `json:"features"`
	DependsOn      []string `json:"depends_on"`
	TargetName     string   `json:"target_name"`
	TargetPath     string   `json:"target_path"`
	RuntimeVersion string   `json:"runtime_version"`
	SourceID       string   `json:"-"`
	SourcePath     string   `json:"-"`
	SourceGroup    string   `json:"-"`
}

// MigrationSourceDetail 正式迁移前重新读取的来源资源详情
type MigrationSourceDetail struct {
	Item      MigrationSourceItem       `json:"item"`
	Website   *MigrationWebsiteDetail   `json:"website,omitempty"`
	Database  *MigrationDatabaseDetail  `json:"database,omitempty"`
	Project   *MigrationProjectDetail   `json:"project,omitempty"`
	Container *MigrationContainerDetail `json:"container,omitempty"`
	Compose   *MigrationComposeDetail   `json:"compose,omitempty"`
}

type MigrationWebsiteDetail struct {
	Type         string              `json:"type"`
	Path         string              `json:"path"`
	Root         string              `json:"root"`
	Listens      []string            `json:"listens"`
	SSLListens   []string            `json:"ssl_listens"`
	Domains      []string            `json:"domains"`
	Index        []string            `json:"index"`
	PHP          uint                `json:"php"`
	Rewrite      string              `json:"rewrite"`
	OpenBasedir  bool                `json:"open_basedir"`
	Remark       string              `json:"remark"`
	ExpireAt     *time.Time          `json:"expire_at"`
	Enabled      bool                `json:"enabled"`
	SSL          bool                `json:"ssl"`
	SSLCert      string              `json:"ssl_cert"`
	SSLKey       string              `json:"ssl_key"`
	SSLProtocols []string            `json:"ssl_protocols"`
	HSTS         bool                `json:"hsts"`
	OCSP         bool                `json:"ocsp"`
	HTTPRedirect bool                `json:"http_redirect"`
	Proxies      []MigrationProxy    `json:"proxies"`
	Redirects    []MigrationRedirect `json:"redirects"`
}

type MigrationProxy struct {
	Location    string            `json:"location"`
	Pass        string            `json:"pass"`
	Host        string            `json:"host"`
	SNI         string            `json:"sni"`
	HTTPVersion string            `json:"http_version"`
	Headers     map[string]string `json:"headers"`
	Replaces    map[string]string `json:"replaces"`
}

type MigrationRedirect struct {
	Type       string `json:"type"`
	From       string `json:"from"`
	To         string `json:"to"`
	KeepURI    bool   `json:"keep_uri"`
	StatusCode int    `json:"status_code"`
}

type MigrationDatabaseDetail struct {
	Type       string `json:"type"`
	Server     string `json:"server"`
	Version    string `json:"version"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Host       string `json:"host"`
	PasswordOK bool   `json:"password_ok"`
}

type MigrationProjectDetail struct {
	Type         ProjectType `json:"type"`
	Version      string      `json:"version"`
	Path         string      `json:"path"`
	WorkingDir   string      `json:"working_dir"`
	ExecStart    string      `json:"exec_start"`
	User         string      `json:"user"`
	Restart      string      `json:"restart"`
	Port         uint        `json:"port"`
	Domains      []string    `json:"domains"`
	Listens      []string    `json:"listens"`
	Environments []KV        `json:"environments"`
	Enabled      bool        `json:"enabled"`
	Running      bool        `json:"running"`
}

type MigrationContainerDetail struct {
	ID              string                     `json:"id"`
	Image           string                     `json:"image"`
	Entrypoint      []string                   `json:"entrypoint"`
	Command         []string                   `json:"command"`
	Env             []KV                       `json:"env"`
	Labels          []KV                       `json:"labels"`
	Ports           []MigrationContainerPort   `json:"ports"`
	Volumes         []MigrationContainerVolume `json:"volumes"`
	Network         string                     `json:"network"`
	NetworkAliases  []string                   `json:"network_aliases"`
	StaticIP        string                     `json:"static_ip"`
	RestartPolicy   string                     `json:"restart_policy"`
	Hostname        string                     `json:"hostname"`
	WorkingDir      string                     `json:"working_dir"`
	User            string                     `json:"user"`
	Privileged      bool                       `json:"privileged"`
	AutoRemove      bool                       `json:"auto_remove"`
	ReadonlyRootfs  bool                       `json:"readonly_rootfs"`
	OpenStdin       bool                       `json:"open_stdin"`
	PublishAllPorts bool                       `json:"publish_all_ports"`
	Tty             bool                       `json:"tty"`
	CPUShares       int64                      `json:"cpu_shares"`
	CPUs            float64                    `json:"cpus"`
	Memory          int64                      `json:"memory"`
	DNS             []string                   `json:"dns"`
	ExtraHosts      []string                   `json:"extra_hosts"`
	CapAdd          []string                   `json:"cap_add"`
	CapDrop         []string                   `json:"cap_drop"`
	Devices         []ContainerDevice          `json:"devices"`
	SecurityOpt     []string                   `json:"security_opt"`
	Sysctls         []KV                       `json:"sysctls"`
	Ulimits         []ContainerUlimit          `json:"ulimits"`
	Tmpfs           []KV                       `json:"tmpfs"`
	ShmSize         int64                      `json:"shm_size"`
	Init            bool                       `json:"init"`
	StopSignal      string                     `json:"stop_signal"`
	StopTimeout     int                        `json:"stop_timeout"`
	Healthcheck     *ContainerHealthcheck      `json:"healthcheck"`
	Running         bool                       `json:"running"`
}

type MigrationContainerPort struct {
	Container uint       `json:"container"`
	Host      netip.Addr `json:"host"`
	HostPort  uint       `json:"host_port"`
	Protocol  string     `json:"protocol"`
}

type MigrationContainerVolume struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	BackupPath  string `json:"backup_path"`
}

type MigrationComposeDetail struct {
	Path         string            `json:"path"`
	Compose      string            `json:"compose"`
	Envs         []KV              `json:"envs"`
	Running      bool              `json:"running"`
	ImageTags    map[string]string `json:"image_tags"`
	ImageSources map[string]string `json:"image_sources"`
}

type MigrationArtifact struct {
	RemotePath  string   `json:"remote_path"`
	RemotePaths []string `json:"remote_paths"`
	FileName    string   `json:"file_name"`
	Kind        string   `json:"kind"`
}
