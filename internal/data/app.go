package data

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/expr-lang/expr"
	"github.com/hashicorp/go-version"
	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/shell"
)

type appRepo struct {
	t    *gotext.Locale
	conf *config.Config
	db   *gorm.DB
	log  *slog.Logger
	api  *api.API
}

func NewAppRepo(i do.Injector) (biz.AppRepo, error) {
	return &appRepo{
		t:    do.MustInvoke[*gotext.Locale](i),
		conf: do.MustInvoke[*config.Config](i),
		db:   do.MustInvoke[*gorm.DB](i),
		log:  do.MustInvoke[*slog.Logger](i),
		api:  api.NewAPI(app.Version, app.Locale),
	}, nil
}

func (r *appRepo) Installed() ([]*biz.App, error) {
	var apps []*biz.App
	if err := r.db.Find(&apps).Error; err != nil {
		return nil, err
	}

	return apps, nil

}

func (r *appRepo) GetInstalled(slug string) (*biz.App, error) {
	installed := new(biz.App)
	if err := r.db.Where("slug = ?", slug).First(installed).Error; err != nil {
		return nil, err
	}

	return installed, nil
}

func (r *appRepo) GetInstalledAll(query string, cond ...string) ([]*biz.App, error) {
	var apps []*biz.App
	if err := r.db.Where(query, cond).Find(&apps).Error; err != nil {
		return nil, err
	}

	return apps, nil
}

func (r *appRepo) ListHomeShow() ([]*biz.App, error) {
	var apps []*biz.App
	if err := r.db.Where("show = ?", true).Order("show_order").Find(&apps).Error; err != nil {
		return nil, err
	}

	return apps, nil
}

func (r *appRepo) IsInstalled(query string, cond ...any) (bool, error) {
	var count int64
	if len(cond) == 0 {
		if err := r.db.Model(&biz.App{}).Where("slug = ?", query).Count(&count).Error; err != nil {
			return false, err
		}
	} else {
		if err := r.db.Model(&biz.App{}).Where(query, cond...).Count(&count).Error; err != nil {
			return false, err
		}
	}

	return count > 0, nil
}

func (r *appRepo) UpdateShow(slug string, show bool) error {
	item, err := r.GetInstalled(slug)
	if err != nil {
		return err
	}

	item.Show = show

	return r.db.Save(item).Error
}

func (r *appRepo) UpdateOrder(slugs []string) error {
	for i, slug := range slugs {
		if err := r.db.Model(&biz.App{}).Where("slug = ?", slug).Update("show_order", i).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *appRepo) DownloadCallback(slug string) {
	// 下载回调
	if err := r.api.AppCallback(slug); err != nil {
		r.log.Warn("download callback failed", slog.String("type", biz.OperationTypeApp), slog.Uint64("operator_id", 0), slog.String("app", slug), slog.Any("err", err))
	}
}

// CheckPanelVersion 校验面板版本号可解析；原编排在存在性检查前先做此校验
func (r *appRepo) CheckPanelVersion() error {
	_, err := version.NewVersion(app.Version)
	return err
}

// ResolveScript 解析渠道并按操作类型生成执行脚本
func (r *appRepo) ResolveScript(item *api.App, matchChannel, action, execVersion string) (string, error) {
	panel, err := version.NewVersion(app.Version)
	if err != nil {
		return "", err
	}

	shellUrl, shellChannel, shellVersion := "", "", ""
	for ch := range slices.Values(item.Channels) {
		vs, err := version.NewVersion(ch.Panel)
		if err != nil {
			continue
		}
		if ch.Slug == matchChannel {
			if vs.GreaterThan(panel) && !r.conf.App.Debug {
				return "", errors.New(r.t.Get("app %s requires panel version %s, current version %s", item.Name, ch.Panel, app.Version))
			}
			switch action {
			case "install":
				shellUrl = fmt.Sprintf("https://%s%s", r.conf.App.DownloadEndpoint, ch.Install)
				shellVersion = ch.Version
			case "uninstall":
				shellUrl = fmt.Sprintf("https://%s%s", r.conf.App.DownloadEndpoint, ch.Uninstall)
				shellVersion = execVersion
			case "update":
				shellUrl = fmt.Sprintf("https://%s%s", r.conf.App.DownloadEndpoint, ch.Update)
				shellVersion = ch.Version
			}
			shellChannel = ch.Slug
			break
		}
	}
	if shellUrl == "" {
		if action == "uninstall" {
			return "", errors.New(r.t.Get("failed to get uninstall script for app %s", item.Name))
		}
		return "", errors.New(r.t.Get("app %s not support current panel version", item.Name))
	}

	return fmt.Sprintf(`curl -sSLm 10 --retry 3 "%s" | bash -s -- "%s" "%s"`, shellUrl, shellChannel, shellVersion), nil
}

// ExecScript 执行脚本
func (r *appRepo) ExecScript(script string) error {
	return shell.ExecfWithOutput(script)
}

func (r *appRepo) PreCheck(app *api.App, catalog api.Apps) error {
	var apps []string
	var installed []string

	all := catalog
	for _, item := range all {
		apps = append(apps, item.Slug)
	}
	installedApps, err := r.Installed()
	if err != nil {
		return err
	}
	for _, item := range installedApps {
		installed = append(installed, item.Slug)
	}

	env := map[string]any{
		"apps":      apps,
		"installed": installed,
	}
	output, err := expr.Eval(app.Depends, env)
	if err != nil {
		return err
	}

	result := cast.ToString(output)
	if result != "ok" {
		return errors.New(r.t.Get("App %s %s", app.Name, result))
	}

	return nil
}
