package bootstrap

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/v3/internal/middleware"
	"github.com/acepanel/panel/v3/pkg/websitestat"
)

// ProviderSet 装配基础设施层
var ProviderSet = wire.NewSet(
	NewConf,
	NewT,
	NewLogger,
	NewSlog,
	NewDB,
	NewMigrate,
	NewSession,
	NewRunner,
	NewValidator,
	middleware.NewMiddlewares,
	NewLoader,
	NewRouter,
	NewTLSReloader,
	NewHttp,
	NewCron,
	NewCli,
	websitestat.NewAggregator,
)
