//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/apps"
	"github.com/acepanel/panel/internal/bootstrap"
	"github.com/acepanel/panel/internal/data"
	"github.com/acepanel/panel/internal/http/middleware"
	"github.com/acepanel/panel/internal/job"
	"github.com/acepanel/panel/internal/route"
	"github.com/acepanel/panel/internal/service"
)

// initWeb init application.
func initWeb() (*app.Web, error) {
	panic(wire.Build(bootstrap.ProviderSet, middleware.ProviderSet, route.ProviderSet, service.ProviderSet, data.ProviderSet, apps.ProviderSet, job.ProviderSet, app.NewWeb))
}
