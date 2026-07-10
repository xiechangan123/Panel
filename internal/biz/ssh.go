package biz

import (
	"context"
	"log/slog"
	"time"

	"github.com/libtnb/utils/crypt"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/ssh"
)

type SSH struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	Name      string           `gorm:"not null;default:''" json:"name"`
	Host      string           `gorm:"not null;default:''" json:"host"`
	Port      uint             `gorm:"not null;default:0" json:"port"`
	Config    ssh.ClientConfig `gorm:"not null;default:'{}';serializer:json" json:"config"`
	Remark    string           `gorm:"not null;default:''" json:"remark"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func (r *SSH) BeforeSave(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	r.Config.Key, err = crypter.Encrypt([]byte(r.Config.Key))
	if err != nil {
		return err
	}
	r.Config.Password, err = crypter.Encrypt([]byte(r.Config.Password))
	if err != nil {
		return err
	}

	return nil
}

func (r *SSH) AfterFind(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	key, err := crypter.Decrypt(r.Config.Key)
	if err == nil {
		r.Config.Key = string(key)
	}
	password, err := crypter.Decrypt(r.Config.Password)
	if err == nil {
		r.Config.Password = string(password)
	}

	return nil
}

type SSHRepo interface {
	List(page, limit uint) ([]*SSH, int64, error)
	Get(id uint) (*SSH, error)
	Create(req *request.SSHCreate) error
	Update(req *request.SSHUpdate) error
	Delete(id uint) error
}

// SSHUsecase SSH 业务逻辑
type SSHUsecase struct {
	repo SSHRepo
	log  *slog.Logger
}

func NewSSHUsecase(repo SSHRepo, log *slog.Logger) *SSHUsecase {
	return &SSHUsecase{repo: repo, log: log}
}

func (uc *SSHUsecase) List(page, limit uint) ([]*SSH, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *SSHUsecase) Get(id uint) (*SSH, error) {
	return uc.repo.Get(id)
}

func (uc *SSHUsecase) Create(ctx context.Context, req *request.SSHCreate) error {
	if err := uc.repo.Create(req); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("ssh created", slog.String("type", OperationTypeSSH), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name), slog.String("host", req.Host))

	return nil
}

func (uc *SSHUsecase) Update(ctx context.Context, req *request.SSHUpdate) error {
	if err := uc.repo.Update(req); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("ssh updated", slog.String("type", OperationTypeSSH), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", req.Name))

	return nil
}

func (uc *SSHUsecase) Delete(ctx context.Context, id uint) error {
	s, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	if err = uc.repo.Delete(id); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("ssh deleted", slog.String("type", OperationTypeSSH), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("name", s.Name))

	return nil
}
