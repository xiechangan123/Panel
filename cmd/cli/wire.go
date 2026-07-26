//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/command"
	"github.com/acepanel/panel/v3/internal/data"
	"github.com/acepanel/panel/v3/internal/service"
)

func initCli() (*app.Cli, func(), error) {
	panic(wire.Build(
		bootstrap.ProviderSet,
		biz.ProviderSet,
		data.ProviderSet,
		service.ProviderSet,
		command.ProviderSet,
		app.NewCli,
	))
}
