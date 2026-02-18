package bootstrap

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/pkg/websitestat"
)

// ProviderSet is bootstrap providers.
var ProviderSet = wire.NewSet(NewConf, NewT, NewLog, NewCli, NewValidator, NewRouter, NewTLSReloader, NewHttp, NewDB, NewMigrate, NewLoader, NewSession, NewCron, NewRunner, websitestat.NewAggregator)
