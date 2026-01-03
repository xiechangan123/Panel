package bootstrap

import (
	"github.com/gookit/validate"
	"github.com/gookit/validate/locales/ruru"
	"github.com/gookit/validate/locales/zhcn"
	"github.com/gookit/validate/locales/zhtw"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/http/rule"
	"github.com/acepanel/panel/pkg/config"
)

// NewValidator just for register global rules
func NewValidator(conf *config.Config, db *gorm.DB) *validate.Validation {
	switch conf.App.Locale {
	case "zh_CN":
		zhcn.RegisterGlobal()
	case "zh_TW":
		zhtw.RegisterGlobal()
	case "ru_RU":
		ruru.RegisterGlobal()
	}

	validate.Config(func(opt *validate.GlobalOption) {
		opt.StopOnError = false
		opt.SkipOnEmpty = true
	})

	// register global rules
	rule.GlobalRules(db)

	return validate.NewEmpty()
}
