package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/v3/internal/request"
)

type SettingKey string

const (
	SettingKeyName                      SettingKey = "name"
	SettingKeyVersion                   SettingKey = "version"
	SettingKeyChannel                   SettingKey = "channel"
	SettingKeyMonitor                   SettingKey = "monitor"
	SettingKeyMonitorDays               SettingKey = "monitor_days"
	SettingKeyMonitorInterval           SettingKey = "monitor_interval"
	SettingKeyBackupPath                SettingKey = "backup_path"
	SettingKeyBackupFormat              SettingKey = "backup_format" // tar.xz / tar.gz / tar.zst / zip / 7z
	SettingKeyWebsitePath               SettingKey = "website_path"
	SettingKeyProjectPath               SettingKey = "project_path"
	SettingKeyContainerSock             SettingKey = "container_sock"
	SettingKeyWebsiteTLSVersions        SettingKey = "website_tls_versions"
	SettingKeyMySQLRootPassword         SettingKey = "mysql_root_password"
	SettingKeyPostgresPassword          SettingKey = "postgres_password"
	SettingKeyMongoDBAdminPassword      SettingKey = "mongodb_admin_password"
	SettingKeyClickHouseDefaultPassword SettingKey = "clickhouse_default_password"
	SettingKeyOfflineMode               SettingKey = "offline_mode"
	SettingKeyAutoUpdate                SettingKey = "auto_update"
	SettingKeyWebserver                 SettingKey = "webserver"
	SettingKeyPublicIPs                 SettingKey = "public_ips"
	SettingHiddenMenu                   SettingKey = "hidden_menu"
	SettingKeyCustomLogo                SettingKey = "custom_logo"
	SettingKeyMemo                      SettingKey = "memo"
	SettingKeyScanAware                 SettingKey = "scan_aware"
	SettingKeyScanAwareDays             SettingKey = "scan_aware_days"
	SettingKeyScanAwareInterfaces       SettingKey = "scan_aware_interfaces"
	SettingKeyScanAwareAutoBlock        SettingKey = "scan_aware_auto_block"
	SettingKeyScanAwareBlockThreshold   SettingKey = "scan_aware_block_threshold"
	SettingKeyScanAwareBlockWindow      SettingKey = "scan_aware_block_window"
	SettingKeyScanAwareBlockDuration    SettingKey = "scan_aware_block_duration" // 小时，0=永久
	SettingKeyScanAwareWhitelist        SettingKey = "scan_aware_whitelist"      // JSON 数组
	SettingKeyWebsiteStatDays           SettingKey = "website_stat_days"
	SettingKeyWebsiteStatErrBufMax      SettingKey = "website_stat_err_buf_max"
	SettingKeyWebsiteStatUVMaxKeys      SettingKey = "website_stat_uv_max_keys"
	SettingKeyWebsiteStatIPMaxKeys      SettingKey = "website_stat_ip_max_keys"
	SettingKeyWebsiteStatDetailMaxKeys  SettingKey = "website_stat_detail_max_keys"
	SettingKeyWebsiteStatBodyEnabled    SettingKey = "website_stat_body_enabled"
	SettingKeyIPDBType                  SettingKey = "ipdb_type" // "" / "custom" / "subscribe"
	SettingKeyIPDBURL                   SettingKey = "ipdb_url"  // 订阅链接
	SettingKeyIPDBPath                  SettingKey = "ipdb_path"
	SettingKeyInfoRan                   SettingKey = "info_ran" // info 命令是否已运行过
)

type Setting struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Key       SettingKey `gorm:"not null;default:'';unique" json:"key"`
	Value     string     `gorm:"not null;default:''" json:"value"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SettingRepo interface {
	Get(key SettingKey, defaultValue ...string) (string, error)
	GetBool(key SettingKey, defaultValue ...bool) (bool, error)
	GetInt(key SettingKey, defaultValue ...int) (int, error)
	GetSlice(key SettingKey, defaultValue ...[]string) ([]string, error)
	Set(key SettingKey, value string) error
	SetSlice(key SettingKey, value []string) error
	Delete(key SettingKey) error
	GetPanel() (*request.SettingPanel, error)
	UpdatePanel(ctx context.Context, req *request.SettingPanel) (bool, error)
	UpdateCert(req *request.SettingCert) error
}

type SettingUsecase struct {
	repo SettingRepo
}

func NewSettingUsecase(repo SettingRepo) *SettingUsecase {
	return &SettingUsecase{repo: repo}
}

func (uc *SettingUsecase) Get(key SettingKey, defaultValue ...string) (string, error) {
	return uc.repo.Get(key, defaultValue...)
}

func (uc *SettingUsecase) GetBool(key SettingKey, defaultValue ...bool) (bool, error) {
	return uc.repo.GetBool(key, defaultValue...)
}

func (uc *SettingUsecase) GetInt(key SettingKey, defaultValue ...int) (int, error) {
	return uc.repo.GetInt(key, defaultValue...)
}

func (uc *SettingUsecase) GetSlice(key SettingKey, defaultValue ...[]string) ([]string, error) {
	return uc.repo.GetSlice(key, defaultValue...)
}

func (uc *SettingUsecase) Set(key SettingKey, value string) error {
	return uc.repo.Set(key, value)
}

func (uc *SettingUsecase) SetSlice(key SettingKey, value []string) error {
	return uc.repo.SetSlice(key, value)
}

func (uc *SettingUsecase) Delete(key SettingKey) error {
	return uc.repo.Delete(key)
}

func (uc *SettingUsecase) GetPanel() (*request.SettingPanel, error) {
	return uc.repo.GetPanel()
}

func (uc *SettingUsecase) UpdatePanel(ctx context.Context, req *request.SettingPanel) (bool, error) {
	return uc.repo.UpdatePanel(ctx, req)
}

func (uc *SettingUsecase) UpdateCert(req *request.SettingCert) error {
	return uc.repo.UpdateCert(req)
}
