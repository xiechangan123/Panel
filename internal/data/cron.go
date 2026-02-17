package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/utils/str"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/io"
	"github.com/acepanel/panel/pkg/os"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/systemctl"
	"github.com/acepanel/panel/pkg/types"
)

type cronRepo struct {
	t   *gotext.Locale
	db  *gorm.DB
	log *slog.Logger
}

func NewCronRepo(t *gotext.Locale, db *gorm.DB, log *slog.Logger) biz.CronRepo {
	return &cronRepo{
		t:   t,
		db:  db,
		log: log,
	}
}

func (r *cronRepo) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&biz.Cron{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *cronRepo) List(page, limit uint) ([]*biz.Cron, int64, error) {
	cron := make([]*biz.Cron, 0)
	var total int64
	err := r.db.Model(&biz.Cron{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&cron).Error
	return cron, total, err
}

func (r *cronRepo) Get(id uint) (*biz.Cron, error) {
	cron := new(biz.Cron)
	if err := r.db.Where("id = ?", id).First(cron).Error; err != nil {
		return nil, err
	}

	return cron, nil
}

func (r *cronRepo) Create(ctx context.Context, req *request.CronCreate) error {
	config := types.CronConfig{
		Type:     req.SubType,
		Flock:    req.Flock,
		Targets:  req.Targets,
		Storage:  req.Storage,
		Keep:     req.Keep,
		URL:      req.URL,
		Method:   req.Method,
		Headers:  req.Headers,
		Body:     req.Body,
		Timeout:  req.Timeout,
		Insecure: req.Insecure,
		Retries:  req.Retries,
	}
	script := r.generateScript(req.Type, config, req.Script)

	shellDir := fmt.Sprintf("%s/server/cron/", app.Root)
	shellLogDir := fmt.Sprintf("%s/server/cron/logs/", app.Root)
	shellFile := str.Random(16)
	if err := io.Write(filepath.Join(shellDir, shellFile+".sh"), script, 0700); err != nil {
		return errors.New(err.Error())
	}
	// 编码转换
	_, _ = shell.Execf("dos2unix %s%s.sh", shellDir, shellFile)

	cron := new(biz.Cron)
	cron.Name = req.Name
	cron.Type = req.Type
	cron.Status = true
	cron.Time = req.Time
	cron.Shell = shellDir + shellFile + ".sh"
	cron.Log = shellLogDir + shellFile + ".log"
	cron.Config = config

	if err := r.db.Create(cron).Error; err != nil {
		return err
	}
	if err := r.addToSystem(cron); err != nil {
		return err
	}

	// 记录日志
	r.log.Info("cron created", slog.String("type", biz.OperationTypeCron), slog.Uint64("operator_id", getOperatorID(ctx)), slog.String("name", req.Name), slog.String("cron_type", req.Type))

	return nil
}

func (r *cronRepo) Update(ctx context.Context, req *request.CronUpdate) error {
	cron, err := r.Get(req.ID)
	if err != nil {
		return err
	}

	cron.Time = req.Time
	cron.Name = req.Name

	// 根据类型重新生成脚本
	if req.Type != "shell" {
		config := types.CronConfig{
			Type:     req.SubType,
			Flock:    req.Flock,
			Targets:  req.Targets,
			Storage:  req.Storage,
			Keep:     req.Keep,
			URL:      req.URL,
			Method:   req.Method,
			Headers:  req.Headers,
			Body:     req.Body,
			Timeout:  req.Timeout,
			Insecure: req.Insecure,
			Retries:  req.Retries,
		}
		cron.Config = config
		script := r.generateScript(req.Type, config, "")
		if err = io.Write(cron.Shell, script, 0700); err != nil {
			return err
		}
	} else {
		cron.Config.Flock = req.Flock
		if err = io.Write(cron.Shell, req.Script, 0700); err != nil {
			return err
		}
	}

	if err = r.db.Save(cron).Error; err != nil {
		return err
	}

	if out, err := shell.Execf("dos2unix %s", cron.Shell); err != nil {
		return errors.New(out)
	}

	if err = r.deleteFromSystem(cron); err != nil {
		return err
	}
	if cron.Status {
		if err = r.addToSystem(cron); err != nil {
			return err
		}
	}

	// 记录日志
	r.log.Info("cron updated", slog.String("type", biz.OperationTypeCron), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", cron.Name))

	return nil
}

func (r *cronRepo) Delete(ctx context.Context, id uint) error {
	cron, err := r.Get(id)
	if err != nil {
		return err
	}

	if err = r.deleteFromSystem(cron); err != nil {
		return err
	}
	if err = io.Remove(cron.Shell); err != nil {
		return err
	}
	// 清理 .lock 文件
	lockFile := strings.TrimSuffix(cron.Shell, ".sh") + ".lock"
	_ = io.Remove(lockFile)

	if err = r.db.Delete(cron).Error; err != nil {
		return err
	}

	// 记录日志
	r.log.Info("cron deleted", slog.String("type", biz.OperationTypeCron), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("name", cron.Name))

	return nil
}

func (r *cronRepo) Status(id uint, status bool) error {
	cron, err := r.Get(id)
	if err != nil {
		return err
	}

	if err = r.deleteFromSystem(cron); err != nil {
		return err
	}
	if status {
		if err = r.addToSystem(cron); err != nil {
			return err
		}
	}

	cron.Status = status

	return r.db.Save(cron).Error
}

// addToSystem 添加到系统
func (r *cronRepo) addToSystem(cron *biz.Cron) error {
	cmd := cron.Shell
	if cron.Config.Flock {
		lockFile := strings.TrimSuffix(cron.Shell, ".sh") + ".lock"
		cmd = fmt.Sprintf("flock -xn %s %s", lockFile, cron.Shell)
	}
	if _, err := shell.Execf(`( crontab -l; echo "%s %s >> %s 2>&1" ) | sort - | uniq - | crontab -`, cron.Time, cmd, cron.Log); err != nil {
		return err
	}

	return r.restartCron()
}

// deleteFromSystem 从系统中删除
func (r *cronRepo) deleteFromSystem(cron *biz.Cron) error {
	if _, err := shell.Execf(`( crontab -l | grep -v -F "%s >> %s 2>&1" ) | crontab -`, cron.Shell, cron.Log); err != nil {
		return err
	}

	return r.restartCron()
}

// restartCron 重启 cron 服务
func (r *cronRepo) restartCron() error {
	if os.IsRHEL() {
		return systemctl.Restart("crond")
	}

	if os.IsDebian() || os.IsUbuntu() {
		return systemctl.Restart("cron")
	}

	return errors.New(r.t.Get("unsupported system"))
}

// generateScript 根据任务类型和配置生成 shell 脚本
func (r *cronRepo) generateScript(typ string, config types.CronConfig, rawScript string) string {
	if typ == "shell" {
		return rawScript
	}

	var sb strings.Builder
	sb.WriteString("#!/bin/bash\nexport PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:$PATH\n\n")

	switch typ {
	case "backup":
		for _, target := range config.Targets {
			switch config.Type {
			case "website":
				_, _ = fmt.Fprintf(&sb, "acepanel backup website -n '%s' -s '%d'\n", target, config.Storage)
			case "mysql", "postgres":
				_, _ = fmt.Fprintf(&sb, "acepanel backup database -t '%s' -n '%s' -s '%d'\n", config.Type, target, config.Storage)
			}
		}
		for _, target := range config.Targets {
			switch config.Type {
			case "website":
				_, _ = fmt.Fprintf(&sb, "acepanel backup clear -t website -f '%s' -k '%d' -s '%d'\n", target, config.Keep, config.Storage)
			case "mysql", "postgres":
				_, _ = fmt.Fprintf(&sb, "acepanel backup clear -t '%s' -f '%s' -k '%d' -s '%d'\n", config.Type, target, config.Keep, config.Storage)
			}
		}
	case "cutoff":
		for _, target := range config.Targets {
			switch config.Type {
			case "website":
				_, _ = fmt.Fprintf(&sb, "acepanel cutoff website -n '%s' -s '%d'\n", target, config.Storage)
			case "container":
				_, _ = fmt.Fprintf(&sb, "acepanel cutoff container -n '%s' -s '%d'\n", target, config.Storage)
			}
		}
		for _, target := range config.Targets {
			switch config.Type {
			case "website":
				_, _ = fmt.Fprintf(&sb, "acepanel cutoff clear -t website -n '%s' -k '%d' -s '%d'\n", target, config.Keep, config.Storage)
			case "container":
				_, _ = fmt.Fprintf(&sb, "acepanel cutoff clear -t container -n '%s' -k '%d' -s '%d'\n", target, config.Keep, config.Storage)
			}
		}
	case "url":
		method := config.Method
		if method == "" {
			method = "GET"
		}
		_, _ = fmt.Fprintf(&sb, "curl -sSL -X %s", method)
		if config.Timeout > 0 {
			_, _ = fmt.Fprintf(&sb, " --connect-timeout %d", config.Timeout)
		}
		if config.Insecure {
			sb.WriteString(" -k")
		}
		if config.Retries > 0 {
			_, _ = fmt.Fprintf(&sb, " --retry %d", config.Retries)
		}
		for key, value := range config.Headers {
			_, _ = fmt.Fprintf(&sb, " -H '%s: %s'", strings.ReplaceAll(key, "'", "'\"'\"'"), strings.ReplaceAll(value, "'", "'\"'\"'"))
		}
		if config.Body != "" {
			_, _ = fmt.Fprintf(&sb, " -d '%s'", strings.ReplaceAll(config.Body, "'", "'\"'\"'"))
		}
		_, _ = fmt.Fprintf(&sb, " '%s'\n", strings.ReplaceAll(config.URL, "'", "'\"'\"'"))
	case "synctime":
		sb.WriteString("acepanel sync-time\n")
	}

	return sb.String()
}
