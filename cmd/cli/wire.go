//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/apps"
	"github.com/acepanel/panel/internal/bootstrap"
	"github.com/acepanel/panel/internal/data"
	"github.com/acepanel/panel/internal/route"
	"github.com/acepanel/panel/internal/service"
)

// initCli init command line.
func initCli() (*app.Cli, error) {
	panic(wire.Build(bootstrap.ProviderSet, route.ProviderSet, service.ProviderSet, data.ProviderSet, apps.ProviderSet, app.NewCli))
}
