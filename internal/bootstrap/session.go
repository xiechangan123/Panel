package bootstrap

import (
	"log/slog"

	"github.com/libtnb/gormstore"
	"github.com/libtnb/sessions"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/pkg/config"
)

func NewSession(i do.Injector) (*sessions.Manager, error) {
	conf := do.MustInvoke[*config.Config](i)

	// initialize session manager
	manager, err := sessions.NewManager(&sessions.ManagerOptions{
		Key:                  conf.App.Key,
		Lifetime:             int(conf.Session.Lifetime),
		GcInterval:           5,
		DisableDefaultDriver: true,
		Logger:               do.MustInvoke[*slog.Logger](i),
	})
	if err != nil {
		return nil, err
	}

	// extend gorm store driver
	store := gormstore.New(do.MustInvoke[*gorm.DB](i))
	if err = manager.Extend("default", store); err != nil {
		return nil, err
	}

	return manager, nil
}
