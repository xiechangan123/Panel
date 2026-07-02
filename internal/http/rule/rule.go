package rule

import (
	"github.com/libtnb/validator"
	"gorm.io/gorm"
)

// RegisterRules 注册面板自定义规则到验证器实例
func RegisterRules(v *validator.Validator, db *gorm.DB) {
	v.RegisterRule(NewExists(db))
	v.RegisterRule(NewNotExists(db))
	v.RegisterRule(NewPassword())
	v.RegisterRule(NewCron())
	v.RegisterRule(NewIPCIDR())
	v.RegisterRule(NewUnixPath())
}
