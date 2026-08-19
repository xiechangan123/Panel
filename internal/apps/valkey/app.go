package valkey

import (
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/apps/redis"
	"github.com/acepanel/panel/v3/internal/biz"
)

type App struct {
	redis *redis.App
}

func NewApp(t *gotext.Locale, databaseServerRepo biz.DatabaseServerRepo, taskRepo biz.TaskRepo) *App {
	return &App{
		redis: redis.New("valkey", "Valkey", t, databaseServerRepo, taskRepo),
	}
}

func (s *App) Route(r chi.Router) {
	s.redis.Route(r)
}

func (s *App) Status() string {
	return s.redis.Status()
}
