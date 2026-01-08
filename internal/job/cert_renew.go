package job

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/io"
	"github.com/acepanel/panel/pkg/systemctl"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	pkgcert "github.com/acepanel/panel/pkg/cert"
)

// CertRenew 证书续签
type CertRenew struct {
	conf            *config.Config
	db              *gorm.DB
	log             *slog.Logger
	settingRepo     biz.SettingRepo
	certRepo        biz.CertRepo
	certAccountRepo biz.CertAccountRepo
}

func NewCertRenew(conf *config.Config, db *gorm.DB, log *slog.Logger, setting biz.SettingRepo, cert biz.CertRepo, certAccount biz.CertAccountRepo) *CertRenew {
	return &CertRenew{
		conf:            conf,
		db:              db,
		log:             log,
		settingRepo:     setting,
		certRepo:        cert,
		certAccountRepo: certAccount,
	}
}

func (r *CertRenew) Run() {
	if app.Status != app.StatusNormal {
		return
	}

	var certs []biz.Cert
	if err := r.db.Preload("Website").Preload("Account").Preload("DNS").Find(&certs).Error; err != nil {
		r.log.Warn("[CertRenew] failed to get certs", slog.Any("err", err))
		return
	}

	for _, cert := range certs {
		if cert.Type == "upload" || !cert.AutoRenew {
			continue
		}

		decode, err := pkgcert.ParseCert(cert.Cert)
		if err != nil {
			continue
		}

		// 结束时间大于 7 天的证书不续签
		if time.Until(decode.NotAfter) > 24*7*time.Hour {
			continue
		}

		_, err = r.certRepo.Renew(cert.ID)
		if err != nil {
			r.log.Warn("[CertRenew] failed to renew cert", slog.Any("err", err))
		}
	}

	// 面板证书续签
	if r.conf.HTTP.ACME {
		decode, err := pkgcert.ParseCert(filepath.Join(app.Root, "panel/storage/cert.pem"))
		if err != nil {
			r.log.Warn("[CertRenew] failed to parse panel cert", slog.Any("err", err))
			return
		}
		// 结束时间大于 2 天不续签
		if time.Until(decode.NotAfter) > 24*2*time.Hour {
			return
		}

		ip, err := r.settingRepo.Get(biz.SettingKeyIP)
		if err != nil || ip == "" {
			r.log.Warn("[CertRenew] failed to get panel IP", slog.Any("err", err))
			return
		}

		var user biz.User
		if err = r.db.First(&user).Error; err != nil {
			r.log.Warn("[CertRenew] failed to get a panel user", slog.Any("err", err))
			return
		}
		account, err := r.certAccountRepo.GetDefault(user.ID)
		if err != nil {
			r.log.Warn("[CertRenew] failed to get panel ACME account", slog.Any("err", err))
			return
		}
		crt, key, err := r.certRepo.ObtainPanel(account, []string{ip})
		if err != nil {
			r.log.Warn("[CertRenew] failed to obtain ACME cert", slog.Any("err", err))
			return
		}

		if err = io.Write(filepath.Join(app.Root, "panel/storage/cert.pem"), string(crt), 0644); err != nil {
			r.log.Warn("[CertRenew] failed to write panel cert", slog.Any("err", err))
			return
		}
		if err = io.Write(filepath.Join(app.Root, "panel/storage/cert.key"), string(key), 0644); err != nil {
			r.log.Warn("[CertRenew] failed to write panel cert key", slog.Any("err", err))
			return
		}

		r.log.Info("[CertRenew] panel cert renewed successfully")
		_ = systemctl.Restart("panel")
	}

}
