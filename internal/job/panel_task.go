package job

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/libtnb/utils/collect"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/tools"
)

// PanelTask 面板每日任务
type PanelTask struct {
	api         *api.API
	conf        *config.Config
	db          *gorm.DB
	log         *slog.Logger
	backupRepo  *biz.BackupUsecase
	cacheRepo   *biz.CacheUsecase
	taskRepo    *biz.TaskUsecase
	settingRepo *biz.SettingUsecase
	scanRepo    *biz.ScanEventUsecase
	statRepo    *biz.WebsiteStatUsecase
	tamperRepo  *biz.TamperUsecase
}

// NewPanelTask 构造面板每日任务
func NewPanelTask(i do.Injector) (Job, error) {
	return Job{
		Spec: "0 2 * * *",
		Task: &PanelTask{
			api:         api.NewAPI(app.Version, app.Locale),
			conf:        do.MustInvoke[*config.Config](i),
			db:          do.MustInvoke[*gorm.DB](i),
			log:         do.MustInvoke[*slog.Logger](i),
			backupRepo:  do.MustInvoke[*biz.BackupUsecase](i),
			cacheRepo:   do.MustInvoke[*biz.CacheUsecase](i),
			taskRepo:    do.MustInvoke[*biz.TaskUsecase](i),
			settingRepo: do.MustInvoke[*biz.SettingUsecase](i),
			scanRepo:    do.MustInvoke[*biz.ScanEventUsecase](i),
			statRepo:    do.MustInvoke[*biz.WebsiteStatUsecase](i),
			tamperRepo:  do.MustInvoke[*biz.TamperUsecase](i),
		},
	}, nil
}

func (r *PanelTask) Run(_ context.Context) error {
	app.Status = app.StatusMaintain

	// 优化主数据库
	if err := r.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
		app.Status = app.StatusFailed
		r.log.Warn("failed to wal checkpoint database", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		return nil
	}
	if err := r.db.Exec("VACUUM").Error; err != nil {
		app.Status = app.StatusFailed
		r.log.Warn("failed to vacuum database", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		return nil
	}
	// 优化辅助数据库
	if err := r.scanRepo.VacuumDB(); err != nil {
		r.log.Warn("failed to vacuum scan event database", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
	}
	if err := r.statRepo.VacuumDB(); err != nil {
		r.log.Warn("failed to vacuum website stat database", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
	}
	if err := r.tamperRepo.VacuumDB(); err != nil {
		r.log.Warn("failed to vacuum tamper database", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
	}

	// 备份面板
	if err := r.backupRepo.CreatePanel(); err != nil {
		r.log.Warn("failed to backup panel", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
	}

	// 清理备份
	if err := r.backupRepo.ClearExpired(r.backupRepo.GetDefaultPath(biz.BackupTypePanel), "panel_", 10); err != nil {
		r.log.Warn("failed to clear backup", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
	}

	// 非离线模式下任务
	if offline, err := r.settingRepo.GetBool(biz.SettingKeyOfflineMode); err == nil && !offline {
		// 更新 IPDB 订阅
		r.updateIPDB()
		// 同步云端数据
		r.updateCategories()
		r.updateApps()
		r.updateEnvironments()
		r.updateTemplates()
		// 自动更新面板
		if autoUpdate, err := r.settingRepo.GetBool(biz.SettingKeyAutoUpdate); err == nil && autoUpdate {
			r.updatePanel()
		}
	}

	// 回收内存
	runtime.GC()
	debug.FreeOSMemory()

	app.Status = app.StatusNormal
	return nil
}

// 更新分类缓存
func (r *PanelTask) updateCategories() {
	time.AfterFunc(time.Duration(rand.IntN(300))*time.Second, func() {
		if err := r.cacheRepo.UpdateCategories(); err != nil {
			r.log.Warn("failed to update categories cache", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}
	})
}

// 更新商店缓存
func (r *PanelTask) updateApps() {
	time.AfterFunc(time.Duration(rand.IntN(300))*time.Second, func() {
		if err := r.cacheRepo.UpdateApps(); err != nil {
			r.log.Warn("failed to update apps cache", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}
	})
}

// 更新运行环境缓存
func (r *PanelTask) updateEnvironments() {
	time.AfterFunc(time.Duration(rand.IntN(300))*time.Second, func() {
		if err := r.cacheRepo.UpdateEnvironments(); err != nil {
			r.log.Warn("failed to update environment cache", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}
	})
}

// 更新模版缓存
func (r *PanelTask) updateTemplates() {
	time.AfterFunc(time.Duration(rand.IntN(300))*time.Second, func() {
		if err := r.cacheRepo.UpdateTemplates(); err != nil {
			r.log.Warn("failed to update template cache", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}
	})
}

// 更新面板
func (r *PanelTask) updatePanel() {
	if r.taskRepo.HasRunningTask() {
		return
	}

	channel, _ := r.settingRepo.Get(biz.SettingKeyChannel)

	// 加 360 秒确保在缓存更新后才更新面板
	time.AfterFunc(time.Duration(rand.IntN(300))*time.Second+360*time.Second, func() {
		panel, err := r.api.LatestVersion(channel)
		if err != nil {
			return
		}
		current, err := version.NewVersion(app.Version)
		if err != nil {
			return
		}
		latest, err := version.NewVersion(panel.Version)
		if err != nil {
			return
		}
		if current.GreaterThanOrEqual(latest) {
			return
		}
		if download := collect.First(panel.Downloads); download != nil {
			url := fmt.Sprintf("https://%s%s", r.conf.App.DownloadEndpoint, download.URL)
			checksum := fmt.Sprintf("https://%s%s", r.conf.App.DownloadEndpoint, download.Checksum)
			if err = r.backupRepo.UpdatePanel(panel.Version, url, checksum, func(msg string) {
				r.log.Info("panel updating", slog.String("type", biz.OperationTypePanel), slog.String("msg", msg))
			}); err != nil {
				r.log.Warn("failed to update panel", slog.String("type", biz.OperationTypePanel), slog.Uint64("operator_id", 0), slog.Any("err", err))
				return
			}
			// 新流程非破坏性、storage 原地不动，失败无需 FixPanel；成功后由本入口负责重启
			tools.RestartPanel()
		}
	})
}

// updateIPDB 更新 IPDB 订阅文件
func (r *PanelTask) updateIPDB() {
	// 文件已存在时每周五更新，不存在则立即下载
	destPath := filepath.Join(app.Root, "panel/storage/geo.ipdb")
	if _, err := os.Stat(destPath); err == nil && time.Now().Weekday() != time.Friday {
		return
	}

	ipdbType, _ := r.settingRepo.Get(biz.SettingKeyIPDBType)
	if ipdbType != "subscribe" {
		return
	}
	ipdbURL, _ := r.settingRepo.Get(biz.SettingKeyIPDBURL)
	if ipdbURL == "" {
		return
	}

	if err := io.DownloadFile(ipdbURL, destPath); err != nil {
		r.log.Warn("failed to download ipdb", slog.String("url", ipdbURL), slog.Any("err", err))
		return
	}

	r.log.Info("ipdb updated", slog.String("url", ipdbURL), slog.String("path", destPath))
}
