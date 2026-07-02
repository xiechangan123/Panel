package bootstrap

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/v3/pkg/websitestat"
)

// ProviderSet is bootstrap providers.
var ProviderSet = wire.NewSet(NewConf, NewT, NewLog, NewCli, NewRouter, NewTLSReloader, NewHttp, NewDB, NewMigrate, NewLoader, NewSession, NewCron, NewRunner, websitestat.NewAggregator)
