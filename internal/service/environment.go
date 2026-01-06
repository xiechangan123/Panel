package service

import (
	"net/http"
	"strings"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type EnvironmentService struct {
	t               *gotext.Locale
	environmentRepo biz.EnvironmentRepo
	taskRepo        biz.TaskRepo
}

func NewEnvironmentService(t *gotext.Locale, environmentRepo biz.EnvironmentRepo, taskRepo biz.TaskRepo) *EnvironmentService {
	return &EnvironmentService{
		t:               t,
		environmentRepo: environmentRepo,
		taskRepo:        taskRepo,
	}
}

func (s *EnvironmentService) Types(w http.ResponseWriter, r *http.Request) {
	Success(w, s.environmentRepo.Types())
}

func (s *EnvironmentService) List(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	all := s.environmentRepo.All()
	var environments []types.EnvironmentDetail
	for _, item := range all {
		if typ != "" && !strings.EqualFold(item.Type, typ) {
			continue
		}
		environments = append(environments, types.EnvironmentDetail{
			Type:        item.Type,
			Name:        item.Name,
			Description: item.Description,
			Slug:        item.Slug,
			Installed:   s.environmentRepo.IsInstalled(item.Type, item.Slug),
			HasUpdate:   s.environmentRepo.HasUpdate(item.Type, item.Slug),
		})
	}

	Success(w, environments)
}

func (s *EnvironmentService) Install(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentAction](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.environmentRepo.Install(req.Type, req.Slug); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *EnvironmentService) Uninstall(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentAction](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.environmentRepo.Uninstall(req.Type, req.Slug); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *EnvironmentService) Update(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentAction](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.environmentRepo.Update(req.Type, req.Slug); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *EnvironmentService) IsInstalled(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentAction](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	installed := s.environmentRepo.IsInstalled(req.Type, req.Slug)
	Success(w, installed)
}
