package biz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/acme"
	"github.com/acepanel/panel/v3/pkg/types"
)

type WebsiteType string

const (
	WebsiteTypeProxy  WebsiteType = "proxy"
	WebsiteTypePHP    WebsiteType = "php"
	WebsiteTypeStatic WebsiteType = "static"
)

type Website struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	Name      string      `gorm:"not null;default:'';unique" json:"name"`
	Type      WebsiteType `gorm:"not null;index;default:'static'" json:"type"`
	Status    bool        `gorm:"not null;default:true" json:"status"`
	Path      string      `gorm:"not null;default:''" json:"path"`
	SSL       bool        `gorm:"not null;default:false" json:"ssl"`
	Remark    string      `gorm:"not null;default:''" json:"remark"`
	ExpireAt  *time.Time  `json:"expire_at"` // 到期时间，nil 表示不限时
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`

	CertExpire string   `gorm:"-:all" json:"cert_expire"` // 仅显示
	PHP        uint     `gorm:"-:all" json:"php"`         // 仅显示
	Domains    []string `gorm:"-:all" json:"domains"`     // 仅显示

	Cert *Cert `gorm:"foreignKey:WebsiteID" json:"cert"`
}

type WebsiteRepo interface {
	GetRewrites() (map[string]string, error)
	UpdateDefaultConfig(req *request.WebsiteDefaultConfig) error
	Count() (int64, error)
	Get(id uint) (*types.WebsiteSetting, error)
	GetByName(name string) (*types.WebsiteSetting, error)
	List(typ string, page, limit uint) ([]*Website, int64, error)
	Create(req *request.WebsiteCreate) (*Website, error)
	Update(req *request.WebsiteUpdate) (*Website, error)
	GetForDelete(id uint) (*Website, error)
	RemoveFiles(name string, removePath bool) error
	Delete(website *Website) error
	ReloadWebServer() error
	UpdateRemark(id uint, remark string) error
	ResetConfig(id uint) error
	UpdateStatus(id uint, status bool) error
	UpdateExpireAt(id uint, expireAt *time.Time) error
	UpdateCert(req *request.WebsiteUpdateCert) error
}

type WebsiteUsecase struct {
	repo           WebsiteRepo
	log            *slog.Logger
	t              *gotext.Locale
	cert           *CertUsecase
	certAccount    *CertAccountUsecase
	database       *DatabaseUsecase
	databaseUser   *DatabaseUserUsecase
	databaseServer DatabaseServerRepo
	tamper         *TamperUsecase
	stat           *WebsiteStatUsecase
}

func NewWebsiteUsecase(i do.Injector) (*WebsiteUsecase, error) {
	return &WebsiteUsecase{
		repo:           do.MustInvoke[WebsiteRepo](i),
		log:            do.MustInvoke[*slog.Logger](i),
		t:              do.MustInvoke[*gotext.Locale](i),
		cert:           do.MustInvoke[*CertUsecase](i),
		certAccount:    do.MustInvoke[*CertAccountUsecase](i),
		database:       do.MustInvoke[*DatabaseUsecase](i),
		databaseUser:   do.MustInvoke[*DatabaseUserUsecase](i),
		databaseServer: do.MustInvoke[DatabaseServerRepo](i),
		tamper:         do.MustInvoke[*TamperUsecase](i),
		stat:           do.MustInvoke[*WebsiteStatUsecase](i),
	}, nil
}

func (uc *WebsiteUsecase) GetRewrites() (map[string]string, error) {
	return uc.repo.GetRewrites()
}

func (uc *WebsiteUsecase) UpdateDefaultConfig(req *request.WebsiteDefaultConfig) error {
	return uc.repo.UpdateDefaultConfig(req)
}

func (uc *WebsiteUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *WebsiteUsecase) Get(id uint) (*types.WebsiteSetting, error) {
	return uc.repo.Get(id)
}

func (uc *WebsiteUsecase) GetByName(name string) (*types.WebsiteSetting, error) {
	return uc.repo.GetByName(name)
}

func (uc *WebsiteUsecase) List(typ string, page, limit uint) ([]*Website, int64, error) {
	return uc.repo.List(typ, page, limit)
}

func (uc *WebsiteUsecase) Create(ctx context.Context, req *request.WebsiteCreate) (*Website, error) {
	w, err := uc.repo.Create(req)
	if err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("website created", slog.String("type", OperationTypeWebsite), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name), slog.String("website_type", req.Type), slog.String("path", req.Path))

	// 重载 Web 服务器
	if err = uc.repo.ReloadWebServer(); err != nil {
		return nil, err
	}

	// 创建数据库
	name := "local_" + req.DBType
	if req.DB {
		server, err := uc.databaseServer.GetByName(name)
		if err != nil {
			return nil, errors.New(uc.t.Get("can't find %s database server, please add it first", name))
		}
		if err = uc.database.Create(ctx, &request.DatabaseCreate{
			ServerID:   server.ID,
			Name:       req.DBName,
			CreateUser: true,
			Username:   req.DBUser,
			Password:   req.DBPassword,
			Host:       "localhost",
			Comment:    fmt.Sprintf("website %s", req.Name),
		}); err != nil {
			return nil, err
		}
	}

	return w, nil
}

func (uc *WebsiteUsecase) Update(ctx context.Context, req *request.WebsiteUpdate) error {
	website, err := uc.repo.Update(req)
	if err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("website updated", slog.String("type", OperationTypeWebsite), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", website.Name))

	return uc.repo.ReloadWebServer()
}

func (uc *WebsiteUsecase) Delete(ctx context.Context, req *request.WebsiteDelete) error {
	website, err := uc.repo.GetForDelete(req.ID)
	if err != nil {
		return err
	}
	if website.Cert != nil {
		return errors.New(uc.t.Get("website %s has bound certificates, please delete the certificate first", website.Name))
	}

	// 清理防篡改规则
	if req.Path && website.Path != "" {
		if rules, listErr := uc.tamper.ListRules(); listErr == nil {
			cleaned := filepath.Clean(website.Path)
			for _, rule := range rules {
				if rule.Path != "" && filepath.Clean(rule.Path) == cleaned {
					_ = uc.tamper.DeleteRule(rule.ID)
				}
			}
		}
	}

	_ = uc.repo.RemoveFiles(website.Name, req.Path)
	if req.DB {
		if mysql, err := uc.databaseServer.GetByName("local_mysql"); err == nil {
			_ = uc.databaseUser.DeleteByNames(mysql.ID, []string{website.Name})
			_ = uc.database.Delete(ctx, mysql.ID, website.Name)
		}
		if postgres, err := uc.databaseServer.GetByName("local_postgresql"); err == nil {
			_ = uc.databaseUser.DeleteByNames(postgres.ID, []string{website.Name})
			_ = uc.database.Delete(ctx, postgres.ID, website.Name)
		}
	}

	if err = uc.repo.Delete(website); err != nil {
		return err
	}

	// 清理该站点在统计库中的全部数据
	_ = uc.stat.DeleteBySite(website.Name)

	// 记录日志
	uc.log.Info("website deleted", slog.String("type", OperationTypeWebsite), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", website.Name))

	return uc.repo.ReloadWebServer()
}

func (uc *WebsiteUsecase) UpdateRemark(id uint, remark string) error {
	return uc.repo.UpdateRemark(id, remark)
}

func (uc *WebsiteUsecase) ResetConfig(id uint) error {
	return uc.repo.ResetConfig(id)
}

func (uc *WebsiteUsecase) UpdateStatus(id uint, status bool) error {
	return uc.repo.UpdateStatus(id, status)
}

func (uc *WebsiteUsecase) UpdateExpireAt(id uint, expireAt *time.Time) error {
	return uc.repo.UpdateExpireAt(id, expireAt)
}

func (uc *WebsiteUsecase) UpdateCert(req *request.WebsiteUpdateCert) error {
	return uc.repo.UpdateCert(req)
}

func (uc *WebsiteUsecase) ObtainCert(ctx context.Context, id uint, dnsID uint) error {
	website, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	// 泛域名必须使用 DNS 验证
	hasWildcard := slices.ContainsFunc(website.Domains, func(d string) bool {
		return strings.Contains(d, "*")
	})
	if hasWildcard && dnsID == 0 {
		return errors.New(uc.t.Get("wildcard domains require DNS verification, please select a DNS provider"))
	}

	account, err := uc.certAccount.GetDefault(cast.ToUint(ctx.Value("user_id")))
	if err != nil {
		return err
	}

	newCert, err := uc.cert.GetByWebsite(website.ID)
	if err != nil {
		if IsNotFound(err) {
			newCert, err = uc.cert.Create(ctx, &request.CertCreate{
				Type:        string(acme.KeyEC256),
				Domains:     website.Domains,
				AutoRenewal: true,
				AccountID:   account.ID,
				DNSID:       dnsID,
				WebsiteID:   website.ID,
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	newCert.Domains = website.Domains
	newCert.DNSID = dnsID
	if err = uc.cert.Save(newCert); err != nil {
		return err
	}

	_, err = uc.cert.ObtainAuto(newCert.ID)
	if err != nil {
		return err
	}

	return uc.cert.Deploy(newCert.ID, website.ID, false)
}
