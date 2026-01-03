package bootstrap

import (
	"github.com/libtnb/gormstore"
	"github.com/libtnb/sessions"
	"gorm.io/gorm"

	"github.com/acepanel/panel/pkg/config"
)

func NewSession(conf *config.Config, db *gorm.DB) (*sessions.Manager, error) {
	// initialize session manager
	manager, err := sessions.NewManager(&sessions.ManagerOptions{
		Key:                  conf.App.Key,
		Lifetime:             int(conf.Session.Lifetime),
		GcInterval:           5,
		DisableDefaultDriver: true,
	})
	if err != nil {
		return nil, err
	}

	// extend gorm store driver
	store := gormstore.New(db)
	if err = manager.Extend("default", store); err != nil {
		return nil, err
	}

	return manager, nil
}
