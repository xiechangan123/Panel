package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type Project struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"not null;unique" json:"name"`                  // 项目名称
	Type      types.ProjectType `gorm:"not null;index;default:'general'" json:"type"` // 项目类型
	Path      string            `gorm:"not null;default:''" json:"path"`              // 项目路径
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type ProjectRepo interface {
	Count() (int64, error)
	List(typ types.ProjectType, page, limit uint) ([]*types.ProjectDetail, int64, error)
	Get(id uint) (*types.ProjectDetail, error)
	Create(ctx context.Context, req *request.ProjectCreate) (*types.ProjectDetail, error)
	Update(ctx context.Context, req *request.ProjectUpdate) error
	Delete(ctx context.Context, id uint) error
}

type ProjectUsecase struct {
	repo ProjectRepo
}

func NewProjectUsecase(repo ProjectRepo) *ProjectUsecase {
	return &ProjectUsecase{repo: repo}
}

func (uc *ProjectUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *ProjectUsecase) List(typ types.ProjectType, page, limit uint) ([]*types.ProjectDetail, int64, error) {
	return uc.repo.List(typ, page, limit)
}

func (uc *ProjectUsecase) Get(id uint) (*types.ProjectDetail, error) {
	return uc.repo.Get(id)
}

func (uc *ProjectUsecase) Create(ctx context.Context, req *request.ProjectCreate) (*types.ProjectDetail, error) {
	return uc.repo.Create(ctx, req)
}

func (uc *ProjectUsecase) Update(ctx context.Context, req *request.ProjectUpdate) error {
	return uc.repo.Update(ctx, req)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}
