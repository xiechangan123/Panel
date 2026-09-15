//go:build wireinject

package main

import (
	"github.com/libtnb/wire"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/data"
	"github.com/acepanel/panel/v3/internal/service"
)

var initCli = wire.New().
	Include(bootstrap.Module, biz.Module, data.Module, service.Module).
	Provide(app.NewCli).
	Injector[func() (*app.Cli, func() error, error)]()
