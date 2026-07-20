package biz

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
)

type FileShare struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Token        string    `gorm:"not null;uniqueIndex" json:"token"`       // 唯一标识（用于下载 URL）
	Path         string    `gorm:"not null;index" json:"path"`              // 文件路径（同一文件可有多条分享）
	Downloads    uint      `gorm:"not null;default:0" json:"downloads"`     // 已下载次数
	MaxDownloads uint      `gorm:"not null;default:0" json:"max_downloads"` // 最大下载次数（0 不限）
	ExpiredAt    time.Time `gorm:"not null;index" json:"expired_at"`        // 过期时间
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FileShareRepo interface {
	List() ([]*FileShare, error)
	Get(id uint) (*FileShare, error)
	Create(path string, maxDownloads uint, expiredAt time.Time) (*FileShare, error)
	Delete(id uint) error
	Consume(token string, count bool) (*FileShare, error)
	ClearExpired() (int64, error)
}

type FileShareUsecase struct {
	repo FileShareRepo
	log  *slog.Logger
}

func NewFileShareUsecase(i do.Injector) (*FileShareUsecase, error) {
	return &FileShareUsecase{
		repo: do.MustInvoke[FileShareRepo](i),
		log:  do.MustInvoke[*slog.Logger](i),
	}, nil
}

func (uc *FileShareUsecase) List() ([]*FileShare, error) {
	return uc.repo.List()
}

// Create 创建分享，同一路径可重复分享为多条独立记录
func (uc *FileShareUsecase) Create(ctx context.Context, req *request.FileShareCreate) (*FileShare, error) {
	share, err := uc.repo.Create(req.Path, req.MaxDownloads, time.Now().Add(time.Duration(req.ExpireHours)*time.Hour))
	if err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("file share created", slog.String("type", OperationTypeFile), slog.Uint64("operator_id", operatorID(ctx)), slog.String("path", req.Path))

	return share, nil
}

func (uc *FileShareUsecase) Delete(ctx context.Context, id uint) error {
	share, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	if err = uc.repo.Delete(id); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("file share deleted", slog.String("type", OperationTypeFile), slog.Uint64("operator_id", operatorID(ctx)), slog.String("path", share.Path))

	return nil
}

// Consume 校验分享有效性，count 为真时原子计数一次下载
func (uc *FileShareUsecase) Consume(token string, count bool) (*FileShare, error) {
	return uc.repo.Consume(token, count)
}

func (uc *FileShareUsecase) ClearExpired() (int64, error) {
	return uc.repo.ClearExpired()
}
