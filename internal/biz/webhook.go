package biz

import (
	"context"
	"log/slog"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/utils/str"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
)

type WebHook struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"not null;default:''" json:"name"`      // 钩子名称
	Key        string    `gorm:"not null;uniqueIndex" json:"key"`      // 唯一标识（用于 URL）
	Script     string    `gorm:"not null;default:''" json:"script"`    // 脚本内容
	Raw        bool      `gorm:"not null;default:false" json:"raw"`    // 是否以原始格式返回输出
	User       string    `gorm:"not null;default:''" json:"user"`      // 以哪个用户身份执行脚本
	Status     bool      `gorm:"not null;default:true" json:"status"`  // 启用状态
	CallCount  uint      `gorm:"not null;default:0" json:"call_count"` // 调用次数
	LastCallAt time.Time `json:"last_call_at"`                         // 上次调用时间
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WebHookRepo interface {
	List(page, limit uint) ([]*WebHook, int64, error)
	Get(id uint) (*WebHook, error)
	GetByKey(key string) (*WebHook, error)
	CreateWithScript(webhook *WebHook, script string) error
	UpdateWithScript(webhook *WebHook, req *request.WebHookUpdate) error
	RemoveScript(key string) error
	Delete(id uint) error
	Call(key string) (string, error)
}

type WebHookUsecase struct {
	repo WebHookRepo
	log  *slog.Logger
	t    *gotext.Locale
}

func NewWebHookUsecase(i do.Injector) (*WebHookUsecase, error) {
	return &WebHookUsecase{
		repo: do.MustInvoke[WebHookRepo](i),
		log:  do.MustInvoke[*slog.Logger](i),
		t:    do.MustInvoke[*gotext.Locale](i),
	}, nil
}

func (uc *WebHookUsecase) List(page, limit uint) ([]*WebHook, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *WebHookUsecase) Get(id uint) (*WebHook, error) {
	return uc.repo.Get(id)
}

func (uc *WebHookUsecase) GetByKey(key string) (*WebHook, error) {
	return uc.repo.GetByKey(key)
}

func (uc *WebHookUsecase) Create(ctx context.Context, req *request.WebHookCreate) (*WebHook, error) {
	key := str.Random(32)
	webhook := &WebHook{
		Name:   req.Name,
		Key:    key,
		Script: req.Script,
		Raw:    req.Raw,
		User:   req.User,
		Status: true,
	}

	if err := uc.repo.CreateWithScript(webhook, req.Script); err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("webhook created", slog.String("type", OperationTypeWebhook), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name))

	return webhook, nil
}

func (uc *WebHookUsecase) Update(ctx context.Context, req *request.WebHookUpdate) error {
	webhook, err := uc.repo.Get(req.ID)
	if err != nil {
		return err
	}

	if err = uc.repo.UpdateWithScript(webhook, req); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("webhook updated", slog.String("type", OperationTypeWebhook), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", req.Name))

	return nil
}

func (uc *WebHookUsecase) Delete(ctx context.Context, id uint) error {
	webhook, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	_ = uc.repo.RemoveScript(webhook.Key)

	if err = uc.repo.Delete(id); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("webhook deleted", slog.String("type", OperationTypeWebhook), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("name", webhook.Name))

	return nil
}

func (uc *WebHookUsecase) Call(key string) (string, error) {
	// 校验与执行紧耦合，留 repo.Call 内单次读取，避免重复读与 TOCTOU
	return uc.repo.Call(key)
}
