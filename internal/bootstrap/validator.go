package bootstrap

import (
	"github.com/libtnb/validator"
	"github.com/libtnb/validator/translations"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/rule"
	"github.com/acepanel/panel/v3/pkg/config"
)

// NewValidator 构建校验器
func NewValidator(i do.Injector) (*validator.Validator, error) {
	conf := do.MustInvoke[*config.Config](i)

	opts := []validator.Option{validator.WithStrictRequired()}
	switch conf.App.Locale {
	case "zh_CN":
		opts = append(opts, validator.WithTranslation(translations.ZhHans()))
	case "zh_TW":
		opts = append(opts, validator.WithTranslation(translations.ZhHant()))
	}

	v := validator.NewValidator(opts...)
	rule.RegisterRules(v, do.MustInvoke[*gorm.DB](i))

	return v, nil
}
