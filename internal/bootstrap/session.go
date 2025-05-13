package bootstrap

import (
	"github.com/go-rat/gormstore"
	"github.com/go-rat/sessions"
	"github.com/knadh/koanf/v2"
	"gorm.io/gorm"
)

func NewSession(conf *koanf.Koanf, db *gorm.DB) (*sessions.Manager, error) {
	// initialize session manager
	manager, err := sessions.NewManager(&sessions.ManagerOptions{
		Key:                  conf.MustString("app.key"),
		Lifetime:             conf.MustInt("session.lifetime"),
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
