package bootstrap

import (
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/embed"
)

func NewT(conf *config.Config) (*gotext.Locale, error) {
	l := gotext.NewLocaleFSWithPath(conf.App.Locale, embed.LocalesFS, "locales")
	l.AddDomain("backend")

	return l, nil
}
