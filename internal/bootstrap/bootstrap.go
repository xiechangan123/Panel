package bootstrap

import (
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/middleware"
	"github.com/acepanel/panel/v3/pkg/websitestat"
)

// Package 装配基础设施层。
var Package = do.Package(
	do.Lazy(NewConf),
	do.Lazy(NewT),
	do.Lazy(NewLogger),
	do.Lazy(NewSlog),
	do.Lazy(NewDB),
	do.Lazy(NewMigrate),
	do.Lazy(NewSession),
	do.Lazy(NewRunner),
	do.Lazy(NewValidator),
	do.Lazy(middleware.NewMiddlewares),
	do.Lazy(NewLoader),
	do.Lazy(NewRouter),
	do.Lazy(NewTLSReloader),
	do.Lazy(NewHttp),
	do.Lazy(NewCron),
	do.Lazy(NewCli),
	do.Lazy(func(i do.Injector) (*websitestat.Aggregator, error) {
		return websitestat.NewAggregator(), nil
	}),
)
