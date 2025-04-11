package bootstrap

import (
	"fmt"
	"slices"

	"github.com/knadh/koanf/v2"
	"github.com/leonelquinteros/gotext"
	"github.com/tnb-labs/panel/pkg/embed"
)

func NewT(conf *koanf.Koanf) (*gotext.Locale, error) {
	dir, err := embed.LocalesFS.ReadDir("locales")
	if err != nil {
		return nil, err
	}
	var locales []string
	for _, d := range dir {
		if d.IsDir() {
			locales = append(locales, d.Name())
		}
	}

	locale := conf.String("app.locale")
	if !slices.Contains(locales, locale) {
		return nil, fmt.Errorf("failed to load locale %s, available locales: %v", locale, locales)
	}

	l := gotext.NewLocaleFSWithPath(locale, embed.LocalesFS, "locales")

	l.AddDomain("web")
	l.AddDomain("cli")

	return l, nil
}
