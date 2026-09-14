package rsync

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"

	"github.com/acepanel/panel/v3/internal/apps/common"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t *gotext.Locale
}

func NewApp(t *gotext.Locale) *App {
	return &App{
		t: t,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/modules", s.List)
	r.Post("/modules", s.Create)
	r.Post("/modules/{name}", s.Update)
	r.Delete("/modules/{name}", s.Delete)
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
}

func (s *App) Status(ctx context.Context) string {
	ok, _ := systemctl.Status(ctx, "rsyncd")
	return types.AggregateAppStatus(ok)
}

func (s *App) List(w http.ResponseWriter, r *http.Request) {
	modules, err := listModules()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := service.Paginate(r, modules)

	service.Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *App) Create(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Module](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if io.Exists(modulePath(req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("module %s already exists", req.Name))
		return
	}

	s.save(r.Context(), w, req)
}

func (s *App) Update(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Module](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if !io.Exists(modulePath(req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("module %s does not exist", req.Name))
		return
	}

	s.save(r.Context(), w, req)
}

func (s *App) Delete(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ModuleName](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	// 删模块与重启不跟随请求取消，否则配置已删而运行中的 rsyncd 仍在提供该模块
	ctx := context.WithoutCancel(r.Context())
	if err = io.Remove(ctx, modulePath(req.Name)); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if err = io.Remove(ctx, secretsPath(req.Name)); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart(ctx, "rsyncd"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	common.ServeConfig(w, rsyncdConf)
}

func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	common.SaveConfig(w, r, rsyncdConf, 0644, "rsyncd")
}

func (s *App) save(ctx context.Context, w http.ResponseWriter, req *Module) {
	if err := writeModule(*req); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err := systemctl.Restart(ctx, "rsyncd"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}
