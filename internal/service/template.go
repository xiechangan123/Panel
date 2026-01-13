package service

import (
	"net/http"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
)

type TemplateService struct {
	t            *gotext.Locale
	templateRepo biz.TemplateRepo
	settingRepo  biz.SettingRepo
}

func NewTemplateService(t *gotext.Locale, template biz.TemplateRepo, setting biz.SettingRepo) *TemplateService {
	return &TemplateService{
		t:            t,
		templateRepo: template,
		settingRepo:  setting,
	}
}

// List 获取所有模版
func (s *TemplateService) List(w http.ResponseWriter, r *http.Request) {
	if offline, _ := s.settingRepo.GetBool(biz.SettingKeyOfflineMode); offline {
		Error(w, http.StatusForbidden, s.t.Get("Unable to get template list in offline mode"))
		return
	}

	templates, err := s.templateRepo.List()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, templates)
}

// Get 获取模版详情
func (s *TemplateService) Get(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TemplateSlug](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if offline, _ := s.settingRepo.GetBool(biz.SettingKeyOfflineMode); offline {
		Error(w, http.StatusForbidden, s.t.Get("Unable to get template in offline mode"))
		return
	}

	template, err := s.templateRepo.Get(req.Slug)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, template)
}

// Create 使用模版创建编排
func (s *TemplateService) Create(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TemplateCreate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if offline, _ := s.settingRepo.GetBool(biz.SettingKeyOfflineMode); offline {
		Error(w, http.StatusForbidden, s.t.Get("Unable to create compose from template in offline mode"))
		return
	}

	// 获取模版
	template, err := s.templateRepo.Get(req.Slug)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 创建编排
	if err = s.templateRepo.CreateCompose(req.Name, template.Compose, req.Envs, req.AutoFirewall); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 回调
	_ = s.templateRepo.Callback(req.Slug)

	Success(w, nil)
}

// Callback 模版下载回调
func (s *TemplateService) Callback(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TemplateSlug](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.templateRepo.Callback(req.Slug); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
