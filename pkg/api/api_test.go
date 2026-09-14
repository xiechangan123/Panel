package api

import (
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func newTestAPI() *API {
	return NewAPI("3.0.0", "en")
}

func TestGetLatestVersion(t *testing.T) {
	version, err := newTestAPI().LatestVersion("stable")
	must.NoError(t, err)
	check.NotEmpty(t, version.Version)
}

func TestGetIntermediateVersions(t *testing.T) {
	// 已是最新版时列表为空是正常的，只能校验不报错
	_, err := newTestAPI().IntermediateVersions("stable")
	check.NoError(t, err)
}

func TestGetCategories(t *testing.T) {
	categories, err := newTestAPI().Categories()
	must.NoError(t, err)
	check.NotEmpty(t, *categories)
}

func TestGetApps(t *testing.T) {
	apps, err := newTestAPI().Apps()
	must.NoError(t, err)
	check.NotEmpty(t, *apps)
}

func TestGetAppBySlug(t *testing.T) {
	app, err := newTestAPI().AppBySlug("nginx")
	must.NoError(t, err)
	check.Equal(t, app.Slug, "nginx")
}

func TestAppCallback(t *testing.T) {
	check.NoError(t, newTestAPI().AppCallback("nginx"))
}

func TestGetTemplates(t *testing.T) {
	templates, err := newTestAPI().Templates()
	must.NoError(t, err)
	check.NotEmpty(t, *templates)
}

func TestGetTemplateBySlug(t *testing.T) {
	template, err := newTestAPI().TemplateBySlug("nginx")
	must.NoError(t, err)
	check.Equal(t, template.Slug, "nginx")
}

func TestTemplateCallback(t *testing.T) {
	check.NoError(t, newTestAPI().TemplateCallback("nginx"))
}
