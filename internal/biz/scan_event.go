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
	Enabled    bool     `json:"enabled"`
	Days       uint     `json:"days"`
	Interfaces []string `json:"interfaces"`
}

// ScanEventRepo 扫描事件数据访问接口
type ScanEventRepo interface {
	Upsert(events []*ScanEvent) error
	List(start, end string, page, limit uint) ([]*ScanEvent, uint, error)
	Summary(start, end string) (*ScanSummary, error)
	Trend(start, end string) ([]*ScanDayTrend, error)
	TopSourceIPs(start, end string, limit uint) ([]*ScanSourceRank, error)
	TopPorts(start, end string, limit uint) ([]*ScanPortRank, error)
	ClearBefore(date string) error
	GetSetting() (*ScanSetting, error)
	UpdateSetting(setting *ScanSetting) error
	Clear() error
}
