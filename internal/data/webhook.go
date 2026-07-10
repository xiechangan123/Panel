package data

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/io"
)

type webhookRepo struct {
	t  *gotext.Locale
	db *gorm.DB
}

func NewWebHookRepo(i do.Injector) (biz.WebHookRepo, error) {
	return &webhookRepo{
		t:  do.MustInvoke[*gotext.Locale](i),
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

func (r *webhookRepo) List(page, limit uint) ([]*biz.WebHook, int64, error) {
	webhooks := make([]*biz.WebHook, 0)
	var total int64
	err := r.db.Model(&biz.WebHook{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&webhooks).Error
	return webhooks, total, err
}

func (r *webhookRepo) Get(id uint) (*biz.WebHook, error) {
	webhook := new(biz.WebHook)
	if err := r.db.Where("id = ?", id).First(webhook).Error; err != nil {
		return nil, err
	}
	return webhook, nil
}

func (r *webhookRepo) GetByKey(key string) (*biz.WebHook, error) {
	webhook := new(biz.WebHook)
	if err := r.db.Where("`key` = ?", key).First(webhook).Error; err != nil {
		return nil, err
	}
	return webhook, nil
}

func (r *webhookRepo) CreateWithScript(webhook *biz.WebHook, script string) error {
	if err := os.MkdirAll(r.webhookDir(), 0755); err != nil {
		return errors.New(r.t.Get("failed to create webhook directory: %v", err))
	}

	scriptFile := r.scriptPath(webhook.Key)
	if err := io.Write(scriptFile, script, 0755); err != nil {
		return errors.New(r.t.Get("failed to write webhook script: %v", err))
	}

	if err := r.db.Create(webhook).Error; err != nil {
		_ = os.Remove(scriptFile)
		return err
	}

	return nil
}

func (r *webhookRepo) UpdateWithScript(webhook *biz.WebHook, req *request.WebHookUpdate) error {
	scriptFile := r.scriptPath(webhook.Key)
	if err := io.Write(scriptFile, req.Script, 0755); err != nil {
		return errors.New(r.t.Get("failed to write webhook script: %v", err))
	}

	if err := r.db.Model(&biz.WebHook{}).Where("id = ?", req.ID).Updates(map[string]any{
		"name":   req.Name,
		"script": req.Script,
		"raw":    req.Raw,
		"user":   req.User,
		"status": req.Status,
	}).Error; err != nil {
		return err
	}

	return nil
}

func (r *webhookRepo) RemoveScript(key string) error {
	scriptFile := r.scriptPath(key)
	_ = os.Remove(scriptFile)
	return nil
}

func (r *webhookRepo) Delete(id uint) error {
	return r.db.Delete(&biz.WebHook{}, id).Error
}

func (r *webhookRepo) Call(key string) (string, error) {
	webhook, err := r.GetByKey(key)
	if err != nil {
		return "", errors.New(r.t.Get("webhook not found"))
	}

	if !webhook.Status {
		return "", errors.New(r.t.Get("webhook is disabled"))
	}

	scriptFile := r.scriptPath(key)
	if !io.Exists(scriptFile) {
		return "", errors.New(r.t.Get("webhook script not found"))
	}

	// 执行脚本
	var cmd *exec.Cmd
	if webhook.User == "" || webhook.User == "root" {
		cmd = exec.Command("bash", scriptFile)
	} else {
		cmd = exec.Command("su", "-s", "/bin/bash", "-c", fmt.Sprintf("bash %s", scriptFile), webhook.User)
	}

	output, err := cmd.CombinedOutput()

	// 更新调用统计
	_ = r.db.Model(&biz.WebHook{}).Where("`key` = ?", key).Updates(map[string]any{
		"call_count":   gorm.Expr("call_count + 1"),
		"last_call_at": time.Now(),
	}).Error

	if err != nil {
		return string(output), fmt.Errorf("script execution failed: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// webhookDir 返回 webhook 脚本存储目录
func (r *webhookRepo) webhookDir() string {
	return filepath.Join(app.Root, "server", "webhook")
}

// scriptPath 返回指定 key 的脚本路径
func (r *webhookRepo) scriptPath(key string) string {
	return filepath.Join(r.webhookDir(), key+".sh")
}
