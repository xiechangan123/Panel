//go:build wireinject

package main

import (
	"github.com/libtnb/wire"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/data"
	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/route"
	"github.com/acepanel/panel/v3/internal/service"
)

var initAce = wire.New().
	Include(bootstrap.Module, apps.Module, biz.Module, data.Module, service.Module, route.Module, job.Module).
	Provide(app.NewAce).
	Injector[func() (*app.Ace, func() error, error)]()
