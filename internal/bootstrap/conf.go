package bootstrap

import (
	"log"
	"time"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/pkg/config"
)

func NewConf(i do.Injector) (*config.Config, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, err
	}

	initGlobal(conf)
	return conf, nil
}

func initGlobal(conf *config.Config) {
	app.Key = conf.App.Key
	if len(app.Key) != 32 {
		log.Fatalf("panel app key must be 32 characters")
	}

	app.Root = "/opt/ace"
	app.Locale = conf.App.Locale

	// 初始化时区
	loc, err := time.LoadLocation(conf.App.Timezone)
	if err != nil {
		log.Fatalf("failed to load timezone: %v", err)
	}
	time.Local = loc
}
