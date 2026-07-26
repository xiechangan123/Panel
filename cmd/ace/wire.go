//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/data"
	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/route"
	"github.com/acepanel/panel/v3/internal/service"
)

func initAce() (*app.Ace, func(), error) {
	panic(wire.Build(
		bootstrap.ProviderSet,
		apps.ProviderSet,
		biz.ProviderSet,
		data.ProviderSet,
		service.ProviderSet,
		route.ProviderSet,
		job.ProviderSet,
		app.NewAce,
	))
}
