package biz

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

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
	List(typ types.ProjectType, page, limit uint) ([]*Project, int64, error)
	GetEntity(id uint) (*Project, error)
	ParseDetail(project *Project) (*types.ProjectDetail, error)
	NameExists(name string) (bool, error)
	Create(project *Project, req *request.ProjectCreate) error
	Save(project *Project) error
	Delete(project *Project) error
	RenameUnitFile(old, new string) error
	RemoveUnitFile(name string) error
	UpdateUnitFile(name string, req *request.ProjectUpdate) error
}

type ProjectUsecase struct {
	repo ProjectRepo
	log  *slog.Logger
	t    *gotext.Locale
}

func NewProjectUsecase(i do.Injector) (*ProjectUsecase, error) {
	return &ProjectUsecase{
		repo: do.MustInvoke[ProjectRepo](i),
		log:  do.MustInvoke[*slog.Logger](i),
		t:    do.MustInvoke[*gotext.Locale](i),
	}, nil
}

func (uc *ProjectUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *ProjectUsecase) List(typ types.ProjectType, page, limit uint) ([]*types.ProjectDetail, int64, error) {
	projects, total, err := uc.repo.List(typ, page, limit)
	if err != nil {
		return nil, 0, err
	}

	details := lo.Map(projects, func(p *Project, _ int) *types.ProjectDetail {
		detail, err := uc.repo.ParseDetail(p)
		if err != nil {
			// 如果解析失败，返回基本信息
			return &types.ProjectDetail{
				ID:   p.ID,
				Name: p.Name,
				Type: p.Type,
			}
		}
		return detail
	})

	return details, total, nil
}

func (uc *ProjectUsecase) Get(id uint) (*types.ProjectDetail, error) {
	project, err := uc.repo.GetEntity(id)
	if err != nil {
		return nil, err
	}
	return uc.repo.ParseDetail(project)
}

func (uc *ProjectUsecase) Create(ctx context.Context, req *request.ProjectCreate) (*types.ProjectDetail, error) {
	// 检查项目名是否已存在
	exists, err := uc.repo.NameExists(req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New(uc.t.Get("project name already exists"))
	}

	project := &Project{
		Name: req.Name,
		Type: req.Type,
		Path: lo.If(!strings.HasPrefix(req.RootDir, "/"), filepath.Join("/", req.RootDir)).Else(req.RootDir),
	}

	if err := uc.repo.Create(project, req); err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("project created", slog.String("type", OperationTypeProject), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name), slog.String("project_type", string(req.Type)))

	return uc.repo.ParseDetail(project)
}

func (uc *ProjectUsecase) Update(ctx context.Context, req *request.ProjectUpdate) error {
	project, err := uc.repo.GetEntity(req.ID)
	if err != nil {
		return err
	}

	// 如果名称变更，需要重命名 unit 文件
	if req.Name != project.Name {
		if err := uc.repo.RenameUnitFile(project.Name, req.Name); err != nil {
			return err
		}
		project.Name = req.Name
	}

	project.Path = lo.If(!strings.HasPrefix(req.RootDir, "/"), filepath.Join("/", req.RootDir)).Else(req.RootDir)
	if err := uc.repo.Save(project); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("project updated", slog.String("type", OperationTypeProject), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", project.Name))

	// 更新 systemd unit 文件
	return uc.repo.UpdateUnitFile(project.Name, req)
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id uint) error {
	project, err := uc.repo.GetEntity(id)
	if err != nil {
		return err
	}

	// 删除 systemd unit 文件
	if err := uc.repo.RemoveUnitFile(project.Name); err != nil {
		return err
	}

	if err := uc.repo.Delete(project); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("project deleted", slog.String("type", OperationTypeProject), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("name", project.Name))

	return nil
}
