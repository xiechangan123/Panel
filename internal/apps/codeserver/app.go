package codeserver

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/acepanel/panel/v3/internal/apps/common"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

const configPath = "/root/.config/code-server/config.yaml"

type App struct{}

func NewApp() *App {
	return &App{}
}

func (s *App) Route(r chi.Router) {
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
}

func (s *App) Status(ctx context.Context) string {
	ok, _ := systemctl.Status(ctx, "code-server")
	return types.AggregateAppStatus(ok)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	common.ServeConfig(w, configPath)
}

func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	common.SaveConfig(w, r, configPath, 0600, "code-server")
}
