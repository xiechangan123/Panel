package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/v3/internal/request"
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
	Create(ctx context.Context, req *request.WebsiteCreate) (*Website, error)
	Update(ctx context.Context, req *request.WebsiteUpdate) error
	Delete(ctx context.Context, req *request.WebsiteDelete) error
	UpdateRemark(id uint, remark string) error
	ResetConfig(id uint) error
	UpdateStatus(id uint, status bool) error
	UpdateExpireAt(id uint, expireAt *time.Time) error
	UpdateCert(req *request.WebsiteUpdateCert) error
	ObtainCert(ctx context.Context, id uint, dnsID uint) error
}

type WebsiteUsecase struct {
	repo WebsiteRepo
}

func NewWebsiteUsecase(repo WebsiteRepo) *WebsiteUsecase {
	return &WebsiteUsecase{repo: repo}
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
	return uc.repo.Create(ctx, req)
}

func (uc *WebsiteUsecase) Update(ctx context.Context, req *request.WebsiteUpdate) error {
	return uc.repo.Update(ctx, req)
}

func (uc *WebsiteUsecase) Delete(ctx context.Context, req *request.WebsiteDelete) error {
	return uc.repo.Delete(ctx, req)
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
	return uc.repo.ObtainCert(ctx, id, dnsID)
}
