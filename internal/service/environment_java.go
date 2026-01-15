package service

import (
	"fmt"
	"net/http"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/shell"
)

type EnvironmentJavaService struct {
	t               *gotext.Locale
	environmentRepo biz.EnvironmentRepo
}

func NewEnvironmentJavaService(t *gotext.Locale, environmentRepo biz.EnvironmentRepo) *EnvironmentJavaService {
	return &EnvironmentJavaService{
		t:               t,
		environmentRepo: environmentRepo,
	}
}

func (s *EnvironmentJavaService) SetCli(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentSlug](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	if !s.environmentRepo.IsInstalled("java", req.Slug) {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("Java-%s is not installed", req.Slug))
		return
	}

	binPath := fmt.Sprintf("%s/server/java/%s/bin", app.Root, req.Slug)
	binaries := []string{"java", "javac", "jar", "jshell"}
	for _, bin := range binaries {
		if _, err = shell.Execf("ln -sf %s/%s /usr/local/bin/%s", binPath, bin, bin); err != nil {
			Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}

	Success(w, nil)
}
