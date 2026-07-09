package biz

import "time"

// ScanEvent 扫描事件模型
type ScanEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SourceIP  string    `gorm:"not null;uniqueIndex:idx_scan_unique" json:"source_ip"`
	Port      uint      `gorm:"not null;uniqueIndex:idx_scan_unique" json:"port"`
	Protocol  string    `gorm:"not null;default:'tcp';uniqueIndex:idx_scan_unique" json:"protocol"`
	Date      string    `gorm:"not null;uniqueIndex:idx_scan_unique;index:idx_scan_date" json:"date"` // YYYY-MM-DD
	Count     uint      `gorm:"not null;default:1" json:"count"`
	Country   string    `gorm:"not null;default:''" json:"country"`
	Region    string    `gorm:"not null;default:''" json:"region"`
	City      string    `gorm:"not null;default:''" json:"city"`
	ISP       string    `gorm:"not null;default:''" json:"isp"`
	FirstSeen time.Time `gorm:"not null" json:"first_seen"`
	LastSeen  time.Time `gorm:"not null" json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ScanSummary 扫描汇总
type ScanSummary struct {
	TotalCount  uint `json:"total_count"`
	UniqueIPs   uint `json:"unique_ips"`
	UniquePorts uint `json:"unique_ports"`
}

// ScanDayTrend 每日趋势
type ScanDayTrend struct {
	Date       string `json:"date"`
	TotalCount uint   `json:"total_count"`
	UniqueIPs  uint   `json:"unique_ips"`
}

// ScanSourceRank 扫描源 IP 排行
type ScanSourceRank struct {
	SourceIP   string `json:"source_ip"`
	TotalCount uint   `json:"total_count"`
	PortCount  uint   `json:"port_count"`
	LastSeen   string `json:"last_seen"`
	Country    string `json:"country"`
	Region     string `json:"region"`
	City       string `json:"city"`
	ISP        string `json:"isp"`
}

// ScanPortRank 被扫描端口排行
type ScanPortRank struct {
	Port       uint   `json:"port"`
	Protocol   string `json:"protocol"`
	TotalCount uint   `json:"total_count"`
	IPCount    uint   `json:"ip_count"`
}

// ScanSetting 扫描感知设置
type ScanSetting struct {
	Enabled        bool     `json:"enabled"`
	Days           uint     `json:"days"`
	Interfaces     []string `json:"interfaces"`
	AutoBlock      bool     `json:"auto_block"`
	BlockThreshold uint     `json:"block_threshold"` // 扫描次数阈值
	BlockWindow    uint     `json:"block_window"`    // 检测窗口（分钟）
	BlockDuration  uint     `json:"block_duration"`  // 屏蔽时长（小时），0=永久
	Whitelist      []string `json:"whitelist"`       // IP/CIDR 白名单
}

// ScanEventRepo 扫描事件数据访问接口
type ScanEventRepo interface {
	Upsert(events []*ScanEvent) error
	List(start, end, sourceIP string, port uint, location string, page, limit uint) ([]*ScanEvent, uint, error)
	Summary(start, end string) (*ScanSummary, error)
	Trend(start, end string) ([]*ScanDayTrend, error)
	TopSourceIPs(start, end string, limit uint) ([]*ScanSourceRank, error)
	TopPorts(start, end string, limit uint) ([]*ScanPortRank, error)
	ClearBefore(date string) error
	GetSetting() (*ScanSetting, error)
	UpdateSetting(setting *ScanSetting) error
	Clear() error
	VacuumDB() error
}

// ScanEventUsecase 扫描事件业务逻辑
type ScanEventUsecase struct {
	repo ScanEventRepo
}

func NewScanEventUsecase(repo ScanEventRepo) *ScanEventUsecase {
	return &ScanEventUsecase{repo: repo}
}

func (uc *ScanEventUsecase) Upsert(events []*ScanEvent) error {
	return uc.repo.Upsert(events)
}

func (uc *ScanEventUsecase) List(start, end, sourceIP string, port uint, location string, page, limit uint) ([]*ScanEvent, uint, error) {
	return uc.repo.List(start, end, sourceIP, port, location, page, limit)
}

func (uc *ScanEventUsecase) Summary(start, end string) (*ScanSummary, error) {
	return uc.repo.Summary(start, end)
}

func (uc *ScanEventUsecase) Trend(start, end string) ([]*ScanDayTrend, error) {
	return uc.repo.Trend(start, end)
}

func (uc *ScanEventUsecase) TopSourceIPs(start, end string, limit uint) ([]*ScanSourceRank, error) {
	return uc.repo.TopSourceIPs(start, end, limit)
}

func (uc *ScanEventUsecase) TopPorts(start, end string, limit uint) ([]*ScanPortRank, error) {
	return uc.repo.TopPorts(start, end, limit)
}

func (uc *ScanEventUsecase) ClearBefore(date string) error {
	return uc.repo.ClearBefore(date)
}

func (uc *ScanEventUsecase) GetSetting() (*ScanSetting, error) {
	return uc.repo.GetSetting()
}

func (uc *ScanEventUsecase) UpdateSetting(setting *ScanSetting) error {
	return uc.repo.UpdateSetting(setting)
}

func (uc *ScanEventUsecase) Clear() error {
	return uc.repo.Clear()
}

func (uc *ScanEventUsecase) VacuumDB() error {
	return uc.repo.VacuumDB()
}
