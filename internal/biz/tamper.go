package biz

import (
	"log/slog"
	"sync"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/pkg/tamper"
)

// TamperRule 防篡改保护规则(通常对应一个网站目录)
type TamperRule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;default:'';unique" json:"name"` // 标识,通常为网站名
	Path      string    `gorm:"not null;default:''" json:"path"`        // 受保护目录
	Exts      []string  `gorm:"serializer:json" json:"exts"`            // 受保护后缀,空=全部
	Excludes  []string  `gorm:"serializer:json" json:"excludes"`        // 排除子路径
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TamperLog 篡改拦截/告警日志
type TamperLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Path      string    `gorm:"not null;default:'';index" json:"path"`
	Op        string    `gorm:"not null;default:''" json:"op"` // write/unlink/rename/create
	PID       uint      `gorm:"not null;default:0" json:"pid"`
	Comm      string    `gorm:"not null;default:''" json:"comm"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TamperSetting 防篡改全局设置
type TamperSetting struct {
	Enabled       bool   `json:"enabled"`
	Mode          string `json:"mode"`            // chattr / ebpf
	BlockNewFiles bool   `json:"block_new_files"` // 新建受保护类型文件时删除拦截
	LogDays       uint   `json:"log_days"`        // 日志保留天数
}

// TamperRepo 防篡改数据访问接口
type TamperRepo interface {
	ListRules() ([]*TamperRule, error)
	GetRule(id uint) (*TamperRule, error)
	GetRuleByName(name string) (*TamperRule, error)
	CreateRule(rule *TamperRule) error
	UpdateRule(rule *TamperRule) error
	DeleteRule(id uint) error
	AddLogs(logs []*TamperLog) error
	ListLogs(page, limit uint) ([]*TamperLog, int64, error)
	ClearLogs() error
	ClearLogsBefore(t time.Time) error
}

// TamperUsecase 防篡改业务逻辑,持有运行时 Manager
type TamperUsecase struct {
	repo    TamperRepo
	setting *SettingUsecase
	log     *slog.Logger

	mu     sync.Mutex
	mgr    *tamper.Manager
	buf    []*TamperLog
	bufMu  sync.Mutex
	drainC chan struct{}
}

func NewTamperUsecase(i do.Injector) (*TamperUsecase, error) {
	return &TamperUsecase{
		repo:    do.MustInvoke[TamperRepo](i),
		setting: do.MustInvoke[*SettingUsecase](i),
		log:     do.MustInvoke[*slog.Logger](i),
	}, nil
}

// Supported 当前平台是否支持防篡改
func (uc *TamperUsecase) Supported() bool {
	return tamper.Supported()
}

// DetectEBPF 检测 eBPF 模式可用性
func (uc *TamperUsecase) DetectEBPF() tamper.EBPFStatus {
	return tamper.DetectEBPF()
}

// EnableBPFLSMGrub 修改 grub 激活 bpf LSM(需重启系统生效)
func (uc *TamperUsecase) EnableBPFLSMGrub() error {
	return tamper.EnableBPFLSMGrub()
}

// GetSetting 读取全局设置
func (uc *TamperUsecase) GetSetting() (*TamperSetting, error) {
	enabled, _ := uc.setting.GetBool(SettingKeyTamperEnabled)
	mode, _ := uc.setting.Get(SettingKeyTamperMode)
	if mode == "" {
		mode = string(tamper.ModeChattr)
	}
	blockNew, _ := uc.setting.GetBool(SettingKeyTamperBlockNew)
	logDays, _ := uc.setting.GetInt(SettingKeyTamperLogDays, 30)
	return &TamperSetting{
		Enabled:       enabled,
		Mode:          mode,
		BlockNewFiles: blockNew,
		LogDays:       uint(logDays),
	}, nil
}

// SaveSetting 保存全局设置并立即生效
func (uc *TamperUsecase) SaveSetting(s *TamperSetting) error {
	if err := uc.setting.Set(SettingKeyTamperMode, s.Mode); err != nil {
		return err
	}
	if err := uc.setting.Set(SettingKeyTamperBlockNew, cast.ToString(s.BlockNewFiles)); err != nil {
		return err
	}
	if err := uc.setting.Set(SettingKeyTamperLogDays, cast.ToString(s.LogDays)); err != nil {
		return err
	}
	if err := uc.setting.Set(SettingKeyTamperEnabled, cast.ToString(s.Enabled)); err != nil {
		return err
	}
	return uc.Reconcile()
}

// Rules 规则管理
func (uc *TamperUsecase) ListRules() ([]*TamperRule, error)    { return uc.repo.ListRules() }
func (uc *TamperUsecase) GetRule(id uint) (*TamperRule, error) { return uc.repo.GetRule(id) }

func (uc *TamperUsecase) CreateRule(rule *TamperRule) error {
	if err := uc.repo.CreateRule(rule); err != nil {
		return err
	}
	return uc.Reconcile()
}

func (uc *TamperUsecase) UpdateRule(rule *TamperRule) error {
	if err := uc.repo.UpdateRule(rule); err != nil {
		return err
	}
	return uc.Reconcile()
}

func (uc *TamperUsecase) DeleteRule(id uint) error {
	if err := uc.repo.DeleteRule(id); err != nil {
		return err
	}
	return uc.Reconcile()
}

// Logs 日志
func (uc *TamperUsecase) ListLogs(page, limit uint) ([]*TamperLog, int64, error) {
	return uc.repo.ListLogs(page, limit)
}
func (uc *TamperUsecase) ClearLogs() error { return uc.repo.ClearLogs() }

// buildConfig 从设置与规则构造运行配置
func (uc *TamperUsecase) buildConfig(s *TamperSetting) (tamper.Config, error) {
	rules, err := uc.repo.ListRules()
	if err != nil {
		return tamper.Config{}, err
	}
	cfg := tamper.Config{
		Mode:          tamper.Mode(s.Mode),
		BlockNewFiles: s.BlockNewFiles,
	}
	for _, r := range rules {
		if !r.Enabled || r.Path == "" {
			continue
		}
		cfg.Rules = append(cfg.Rules, tamper.Rule{
			Name:     r.Name,
			Paths:    []string{r.Path},
			Exts:     r.Exts,
			Excludes: r.Excludes,
		})
	}
	return cfg, nil
}

// Reconcile 依据设置启停并同步 Manager,应由设置/规则变更后调用
func (uc *TamperUsecase) Reconcile() error {
	s, err := uc.GetSetting()
	if err != nil {
		return err
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	// 未启用或无可用规则:停止
	if !s.Enabled {
		uc.stopLocked()
		return nil
	}

	cfg, err := uc.buildConfig(s)
	if err != nil {
		return err
	}
	if len(cfg.Rules) == 0 {
		uc.stopLocked()
		return nil
	}

	// 重启 Manager 以套用最新配置(规则/模式可能变化)
	uc.stopLocked()
	mgr, err := tamper.NewManager(cfg, uc.log)
	if err != nil {
		return err
	}
	if err = mgr.Start(); err != nil {
		_ = mgr.Stop()
		return err
	}
	uc.mgr = mgr
	uc.drainC = make(chan struct{})
	go uc.drain(mgr, uc.drainC)
	return nil
}

// stopLocked 停止当前 Manager(调用方持锁)
func (uc *TamperUsecase) stopLocked() {
	if uc.mgr != nil {
		if uc.drainC != nil {
			close(uc.drainC)
			uc.drainC = nil
		}
		_ = uc.mgr.Stop()
		uc.mgr = nil
	}
}

// drain 持续读取拦截事件缓冲到内存
func (uc *TamperUsecase) drain(mgr *tamper.Manager, done chan struct{}) {
	ch := mgr.Events()
	for {
		select {
		case <-done:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			uc.bufMu.Lock()
			uc.buf = append(uc.buf, &TamperLog{
				Path:      ev.Path,
				Op:        ev.OpStr,
				PID:       uint(ev.PID),
				Comm:      ev.Comm,
				CreatedAt: ev.Time,
			})
			uc.bufMu.Unlock()
		}
	}
}

// FlushLogs 将缓冲的拦截日志落库,由定时任务调用
func (uc *TamperUsecase) FlushLogs() {
	uc.bufMu.Lock()
	if len(uc.buf) == 0 {
		uc.bufMu.Unlock()
		return
	}
	logs := uc.buf
	uc.buf = nil
	uc.bufMu.Unlock()

	if err := uc.repo.AddLogs(logs); err != nil {
		uc.log.Warn("防篡改日志落库失败", slog.Any("err", err))
	}
}

// CleanupLogs 清理过期日志
func (uc *TamperUsecase) CleanupLogs() {
	s, err := uc.GetSetting()
	if err != nil || s.LogDays == 0 {
		return
	}
	_ = uc.repo.ClearLogsBefore(time.Now().AddDate(0, 0, -int(s.LogDays)))
}

// Running 当前是否有 Manager 在运行
func (uc *TamperUsecase) Running() bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.mgr != nil
}

// Stats 运行统计
func (uc *TamperUsecase) Stats() tamper.Stats {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.mgr == nil {
		return tamper.Stats{}
	}
	return uc.mgr.Stats()
}

// Unlock 临时解除指定路径保护(供面板合法写入前调用),返回是否处于保护中
func (uc *TamperUsecase) Unlock(paths ...string) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.mgr == nil {
		return false
	}
	uc.mgr.Unlock(paths)
	return true
}

// Relock 恢复指定路径保护
func (uc *TamperUsecase) Relock(paths ...string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	if uc.mgr != nil {
		uc.mgr.Relock(paths)
	}
}
