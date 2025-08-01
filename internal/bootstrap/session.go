package bootstrap

import (
	"github.com/knadh/koanf/v2"
	"github.com/libtnb/gormstore"
	"github.com/libtnb/sessions"
	"gorm.io/gorm"
)

func NewSession(conf *koanf.Koanf, db *gorm.DB) (*sessions.Manager, error) {
	// initialize session manager
	lifetime := conf.Int("session.lifetime")
	// TODO: will remove this fallback in v3
	if lifetime == 0 {
		lifetime = 120
	}
	manager, err := sessions.NewManager(&sessions.ManagerOptions{
		Key:                  conf.MustString("app.key"),
		Lifetime:             lifetime,
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
