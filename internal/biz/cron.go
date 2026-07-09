package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type Cron struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	Name      string           `gorm:"not null;default:'';unique" json:"name"`
	Status    bool             `gorm:"not null;default:false" json:"status"`
	Type      string           `gorm:"not null;default:''" json:"type"`
	Time      string           `gorm:"not null;default:''" json:"time"`
	Config    types.CronConfig `gorm:"serializer:json;not null;default:'{}'" json:"config"`
	Shell     string           `gorm:"not null;default:''" json:"shell"`
	Log       string           `gorm:"not null;default:''" json:"log"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type CronRepo interface {
	Count() (int64, error)
	List(page, limit uint) ([]*Cron, int64, error)
	Get(id uint) (*Cron, error)
	Create(ctx context.Context, req *request.CronCreate) error
	Update(ctx context.Context, req *request.CronUpdate) error
	Delete(ctx context.Context, id uint) error
	Status(id uint, status bool) error
}

// CronUsecase 计划任务业务逻辑
type CronUsecase struct {
	repo CronRepo
}

func NewCronUsecase(repo CronRepo) *CronUsecase {
	return &CronUsecase{repo: repo}
}

func (uc *CronUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *CronUsecase) List(page, limit uint) ([]*Cron, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *CronUsecase) Get(id uint) (*Cron, error) {
	return uc.repo.Get(id)
}

func (uc *CronUsecase) Create(ctx context.Context, req *request.CronCreate) error {
	return uc.repo.Create(ctx, req)
}

func (uc *CronUsecase) Update(ctx context.Context, req *request.CronUpdate) error {
	return uc.repo.Update(ctx, req)
}

func (uc *CronUsecase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *CronUsecase) Status(id uint, status bool) error {
	return uc.repo.Status(id, status)
}
