package injector

import (
	"time"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/command"
	"github.com/acepanel/panel/v3/internal/data"
	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/route"
	"github.com/acepanel/panel/v3/internal/service"
)

func New() do.Injector {
	return do.NewWithOpts(&do.InjectorOpts{
		HealthCheckGlobalTimeout: 5 * time.Second,
	},
		bootstrap.Package,
		biz.Package,
		data.Package,
		service.Package,
		apps.Package,
		route.Package,
		command.Package,
		job.Package,
		do.Lazy(app.NewAce),
		do.Lazy(app.NewCli),
	)
}
