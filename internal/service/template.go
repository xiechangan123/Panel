package service

import (
	"net/http"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix"

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
	paged, total := Paginate(r, s.templateRepo.List())
	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

// Get 获取模版详情
func (s *TemplateService) Get(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TemplateSlug](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
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

	// 获取模版
	template, err := s.templateRepo.Get(req.Slug)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 创建编排
	dir, err := s.templateRepo.CreateCompose(req.Name, template.Compose, req.Envs, req.AutoFirewall)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 回调
	_ = s.templateRepo.Callback(req.Slug)

	Success(w, dir)
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
