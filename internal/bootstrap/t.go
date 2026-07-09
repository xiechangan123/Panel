package bootstrap

import (
	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/embed"
)

func NewT(i do.Injector) (*gotext.Locale, error) {
	conf := do.MustInvoke[*config.Config](i)

	l := gotext.NewLocaleFSWithPath(conf.App.Locale, embed.LocalesFS, "locales")
	l.AddDomain("backend")

	return l, nil
}
