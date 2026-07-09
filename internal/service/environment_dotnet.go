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

type EnvironmentDotnetService struct {
	t               *gotext.Locale
	environmentRepo *biz.EnvironmentUsecase
}

func NewEnvironmentDotnetService(i do.Injector) (*EnvironmentDotnetService, error) {
	return &EnvironmentDotnetService{
		t:               do.MustInvoke[*gotext.Locale](i),
		environmentRepo: do.MustInvoke[*biz.EnvironmentUsecase](i),
	}, nil
}

func (s *EnvironmentDotnetService) SetCli(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.EnvironmentSlug](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	if !s.environmentRepo.IsInstalled("dotnet", req.Slug) {
		Error(w, http.StatusUnprocessableEntity, s.t.Get(".NET-%s is not installed", req.Slug))
		return
	}

	binPath := fmt.Sprintf("%s/server/dotnet/%s", app.Root, req.Slug)
	if err = io.LinkCLIBinaries(binPath, []string{"dotnet"}); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
