package minio

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/acepanel/panel/v3/internal/apps/common"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

const envPath = "/etc/default/minio"

type App struct{}

func NewApp() *App {
	return &App{}
}

func (s *App) Route(r chi.Router) {
	r.Get("/env", s.GetEnv)
	r.Post("/env", s.UpdateEnv)
}

func (s *App) Status(ctx context.Context) string {
	ok, _ := systemctl.Status(ctx, "minio")
	return types.AggregateAppStatus(ok)
}

func (s *App) GetEnv(w http.ResponseWriter, r *http.Request) {
	common.ServeConfig(w, envPath)
}

func (s *App) UpdateEnv(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateEnv](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	common.WriteConfig(w, r, envPath, req.Env, 0600, "minio")
}
