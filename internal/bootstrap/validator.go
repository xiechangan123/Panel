package bootstrap

import (
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/validator"
	"github.com/libtnb/validator/translations"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/rule"
	"github.com/acepanel/panel/v3/pkg/config"
)

// NewValidator 构建校验器
func NewValidator(conf *config.Config, db *gorm.DB, t *gotext.Locale) *validator.Validator {
	opts := []validator.Option{
		validator.WithStrictRequired(),
		validator.WithRules(rule.Rules(t)...),
		validator.WithFallibleRules(rule.FallibleRules(db, t)...),
		validator.WithMessages(rule.Messages(t)),
		validator.WithAttributes(request.Attributes(t)),
	}
	switch conf.App.Locale {
	case "zh_CN":
		opts = append(opts, validator.WithTranslation(translations.ZhHans()))
	case "zh_TW":
		opts = append(opts, validator.WithTranslation(translations.ZhHant()))
	}

	return validator.MustNew(opts...)
}
