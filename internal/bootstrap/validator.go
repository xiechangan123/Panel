package bootstrap

import (
	"github.com/libtnb/validator"
	"github.com/libtnb/validator/translations"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/rule"
	"github.com/acepanel/panel/v3/pkg/config"
)

// NewValidator 构建校验器
func NewValidator(conf *config.Config, db *gorm.DB) *validator.Validator {

	opts := []validator.Option{validator.WithStrictRequired()}
	switch conf.App.Locale {
	case "zh_CN":
		opts = append(opts, validator.WithTranslation(translations.ZhHans()))
	case "zh_TW":
		opts = append(opts, validator.WithTranslation(translations.ZhHant()))
	}

	v := validator.NewValidator(opts...)
	rule.RegisterRules(v, db)

	return v
}
