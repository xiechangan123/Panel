package service

import (
	"fmt"
	"net/http"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/io"
)

type EnvironmentJavaService struct {
	t               *gotext.Locale
	environmentRepo *biz.EnvironmentUsecase
}

func NewEnvironmentJavaService(i do.Injector) (*EnvironmentJavaService, error) {
	return &EnvironmentJavaService{
		t:               do.MustInvoke[*gotext.Locale](i),
		environmentRepo: do.MustInvoke[*biz.EnvironmentUsecase](i),
	}, nil
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
	if err = io.LinkCLIBinaries(binPath, []string{"java", "javac", "jar", "jshell"}); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
