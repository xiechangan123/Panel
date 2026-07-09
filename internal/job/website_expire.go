package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/libtnb/cron"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

// WebsiteExpire 网站到期自动关闭任务
type WebsiteExpire struct {
	db          *gorm.DB
	log         *slog.Logger
	websiteRepo *biz.WebsiteUsecase
}

// NewWebsiteExpire 创建网站到期检查任务
func NewWebsiteExpire(db *gorm.DB, log *slog.Logger, websiteRepo *biz.WebsiteUsecase) *WebsiteExpire {
	return &WebsiteExpire{
		db:          db,
		log:         log,
		websiteRepo: websiteRepo,
	}
}

func (r *WebsiteExpire) Run(_ context.Context) error {
	if app.Status != app.StatusNormal {
		return nil
	}

	var websites []biz.Website
	now := time.Now()
	// 直接查询已到期且仍在运行的网站
	if err := r.db.Where("expire_at IS NOT NULL AND expire_at <= ? AND status = ?", now, true).Find(&websites).Error; err != nil {
		r.log.Warn("failed to query expired websites", slog.Any("err", err))
		return nil
	}

	for _, website := range websites {
		if err := r.websiteRepo.UpdateStatus(website.ID, false); err != nil {
			r.log.Warn("failed to disable expired website", slog.String("name", website.Name), slog.Any("err", err))
			continue
		}
		r.log.Info("website expired and disabled", slog.String("name", website.Name), slog.Time("expire_at", *website.ExpireAt))
	}
	return nil
}

// WebsiteExpireJob 注册网站到期检查任务
func WebsiteExpireJob(i do.Injector) (JobFn, error) {
	db := do.MustInvoke[*gorm.DB](i)
	log := do.MustInvoke[*slog.Logger](i)
	website := do.MustInvoke[*biz.WebsiteUsecase](i)
	return func(c *cron.Cron) error {
		_, err := c.Add("* * * * *", NewWebsiteExpire(db, log, website))
		return err
	}, nil
}
