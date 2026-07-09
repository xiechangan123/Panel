package job

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/libtnb/cron"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	pkgcert "github.com/acepanel/panel/v3/pkg/cert"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/tools"
)

// CertRenew 证书续签
type CertRenew struct {
	conf            *config.Config
	db              *gorm.DB
	log             *slog.Logger
	settingRepo     *biz.SettingUsecase
	certRepo        *biz.CertUsecase
	certAccountRepo *biz.CertAccountUsecase
}

func NewCertRenew(conf *config.Config, db *gorm.DB, log *slog.Logger, setting *biz.SettingUsecase, cert *biz.CertUsecase, certAccount *biz.CertAccountUsecase) *CertRenew {
	return &CertRenew{
		conf:            conf,
		db:              db,
		log:             log,
		settingRepo:     setting,
		certRepo:        cert,
		certAccountRepo: certAccount,
	}
}

func (r *CertRenew) Run(_ context.Context) error {
	if app.Status != app.StatusNormal {
		return nil
	}

	var certs []biz.Cert
	if err := r.db.Preload("Website").Preload("Account").Preload("DNS").Find(&certs).Error; err != nil {
		r.log.Warn("failed to get certs", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
		return nil
	}

	for _, cert := range certs {
		// 跳过上传类型或未开启自动续签的证书
		if cert.Type == "upload" || !cert.AutoRenewal {
			continue
		}

		// 刷新续签信息
		if cert.RenewalInfo.NeedsRefresh() {
			renewInfo, err := r.certRepo.RefreshRenewalInfo(cert.ID)
			if err != nil {
				r.log.Warn("failed to refresh renewal info", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
				continue
			}
			cert.RenewalInfo = renewInfo
		}

		// 到达建议时间，续签证书
		if time.Now().After(cert.RenewalInfo.SelectedTime) {
			if _, err := r.certRepo.Renew(cert.ID); err != nil {
				r.log.Warn("failed to renew certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			}
		}
	}

	// 面板证书续签
	switch r.conf.HTTP.TLS {
	case "self-signed":
		// 自签证书续签
		crt, _ := os.ReadFile(filepath.Join(app.Root, "panel/storage/cert.pem"))
		decode, err := pkgcert.ParseCert(crt)
		if err == nil {
			if time.Until(decode.NotAfter) > 30*24*time.Hour {
				return nil
			}
		} else {
			r.log.Warn("failed to parse panel certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}

		newCrt, newKey, err := pkgcert.GenerateSelfSigned(tools.CollectLocalNames())
		if err != nil {
			r.log.Warn("failed to generate self-signed certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}
		if err = r.settingRepo.UpdateCert(&request.SettingCert{
			Cert: string(newCrt),
			Key:  string(newKey),
		}); err != nil {
			r.log.Warn("failed to update panel certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}
		r.log.Info("panel self-signed certificate renewed", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0))

	case "off", "custom":
		// off/custom 不需要自动续签

	default:
		// ACME 模式
		crt, _ := os.ReadFile(filepath.Join(app.Root, "panel/storage/cert.pem"))
		decode, err := pkgcert.ParseCert(crt)
		if err == nil {
			// 结束时间大于 2 天不续签
			if time.Until(decode.NotAfter) > 24*2*time.Hour {
				return nil
			}
		} else {
			// 解析失败则继续续签流程，可能是证书格式不对或者文件不存在
			r.log.Warn("failed to parse panel certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
		}

		ip, err := r.settingRepo.Get(biz.SettingKeyPublicIPs)
		if err != nil {
			r.log.Warn("failed to get panel IP", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}
		var ips []string
		if err = json.Unmarshal([]byte(ip), &ips); err != nil || len(ips) == 0 {
			r.log.Warn("panel public IPs not set", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}

		var user biz.User
		if err = r.db.First(&user).Error; err != nil {
			r.log.Warn("failed to get a panel user", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}
		account, err := r.certAccountRepo.GetDefault(user.ID)
		if err != nil {
			r.log.Warn("failed to get panel ACME account", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}
		crt, key, err := r.certRepo.ObtainPanel(account, ips)
		if err != nil {
			r.log.Warn("failed to obtain panel certificate via ACME", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}

		if err = r.settingRepo.UpdateCert(&request.SettingCert{
			Cert: string(crt),
			Key:  string(key),
		}); err != nil {
			r.log.Warn("failed to update panel certificate", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0), slog.Any("err", err))
			return nil
		}

		r.log.Info("panel certificate renewed successfully", slog.String("type", biz.OperationTypeCert), slog.Uint64("operator_id", 0))
	}

	return nil
}

// CertRenewJob 注册证书续签任务
func CertRenewJob(i do.Injector) (JobFn, error) {
	conf := do.MustInvoke[*config.Config](i)
	db := do.MustInvoke[*gorm.DB](i)
	log := do.MustInvoke[*slog.Logger](i)
	setting := do.MustInvoke[*biz.SettingUsecase](i)
	cert := do.MustInvoke[*biz.CertUsecase](i)
	certAccount := do.MustInvoke[*biz.CertAccountUsecase](i)
	return func(c *cron.Cron) error {
		_, err := c.Add("0 4 * * *", NewCertRenew(conf, db, log, setting, cert, certAccount))
		return err
	}, nil
}
