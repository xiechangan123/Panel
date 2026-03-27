package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/utils/str"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/http/request"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/os"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
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
	// 清理 .lock 文件和 _wrapper.sh 文件
	lockFile := strings.TrimSuffix(cron.Shell, ".sh") + ".lock"
	_ = io.Remove(lockFile)
	wrapperFile := strings.TrimSuffix(cron.Shell, ".sh") + "_wrapper.sh"
	_ = io.Remove(wrapperFile)

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

	// 秒级任务：生成 wrapper 脚本，用每分钟触发模拟秒级执行
	if seconds := r.parseSeconds(cron.Time); seconds > 0 {
		wrapperPath := strings.TrimSuffix(cron.Shell, ".sh") + "_wrapper.sh"
		wrapperScript := r.generateWrapper(cmd, cron.Log, seconds)
		if err := io.Write(wrapperPath, wrapperScript, 0700); err != nil {
			return err
		}
		if _, err := shell.Execf(`( crontab -l; echo "* * * * * %s" ) | sort - | uniq - | crontab -`, wrapperPath); err != nil {
			return err
		}
		return r.restartCron()
	}

	if _, err := shell.Execf(`( crontab -l; echo "%s %s >> %s 2>&1" ) | sort - | uniq - | crontab -`, cron.Time, cmd, cron.Log); err != nil {
		return err
	}

	return r.restartCron()
}

// deleteFromSystem 从系统中删除
func (r *cronRepo) deleteFromSystem(cron *biz.Cron) error {
	// 清理秒级任务的 wrapper 条目和脚本
	wrapperPath := strings.TrimSuffix(cron.Shell, ".sh") + "_wrapper.sh"
	_, _ = shell.Execf(`( crontab -l | grep -v -F "%s" ) | crontab -`, wrapperPath)
	_ = io.Remove(wrapperPath)

	// 清理普通任务的 crontab 条目
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
			case "mysql", "postgresql":
				_, _ = fmt.Fprintf(&sb, "acepanel backup database -t '%s' -n '%s' -s '%d'\n", config.Type, target, config.Storage)
			}
		}
		for _, target := range config.Targets {
			switch config.Type {
			case "website":
				_, _ = fmt.Fprintf(&sb, "acepanel backup clear -t website -f '%s' -k '%d' -s '%d'\n", target, config.Keep, config.Storage)
			case "mysql", "postgresql":
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

// parseSeconds 从 6 字段 cron 表达式中解析秒级间隔
// 返回 0 表示非秒级任务
func (r *cronRepo) parseSeconds(time string) int {
	fields := strings.Fields(time)
	if len(fields) != 6 {
		return 0
	}

	// 6 字段格式：秒 分 时 日 月 周，后 5 个字段必须全是 *
	for _, f := range fields[1:] {
		if f != "*" {
			return 0
		}
	}

	second := fields[0]
	// 每秒：* * * * * *
	if second == "*" {
		return 1
	}
	// 每 N 秒：*/N * * * * *
	if strings.HasPrefix(second, "*/") {
		n, err := strconv.Atoi(second[2:])
		if err != nil || n <= 0 || n > 59 {
			return 0
		}
		return n
	}

	return 0
}

// generateWrapper 生成秒级任务的 wrapper 脚本
// 通过每分钟触发 + 循环 sleep 模拟秒级执行
func (r *cronRepo) generateWrapper(cmd, logFile string, seconds int) string {
	count := 60 / seconds
	return fmt.Sprintf(`#!/bin/bash
INTERVAL=%d
COUNT=%d
for i in $(seq 1 $COUNT); do
    %s >> %s 2>&1 &
    [ $i -lt $COUNT ] && sleep $INTERVAL
done
wait
`, seconds, count, cmd, logFile)
}
