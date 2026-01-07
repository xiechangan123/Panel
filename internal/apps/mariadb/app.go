package mariadb

import (
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/apps/mysql"
	"github.com/acepanel/panel/internal/biz"
)

type App struct {
	mysql *mysql.App
}

func NewApp(t *gotext.Locale, setting biz.SettingRepo) *App {
	return &App{
		mysql: mysql.NewApp(t, setting),
	}
}

func (s *App) Route(r chi.Router) {
	s.mysql.Route(r)
}
