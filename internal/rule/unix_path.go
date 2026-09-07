package rule

import (
	"regexp"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/validator"
	"github.com/spf13/cast"
)

// UnixPath 校验 Unix 绝对路径
type UnixPath struct {
	t  *gotext.Locale
	re *regexp.Regexp
}

func NewUnixPath(t *gotext.Locale) *UnixPath {
	return &UnixPath{
		t:  t,
		re: regexp.MustCompile(`^/$|^(/[^/\x00]+)+/?$`),
	}
}

func (r *UnixPath) Signature() string { return "unix_path" }

func (r *UnixPath) Message() string { return r.t.Get("{field} must be a valid Unix absolute path") }

func (r *UnixPath) Passes(f *validator.Field) bool {
	if validator.IsEmptyValue(f.Reflect()) {
		return true
	}
	return r.re.MatchString(cast.ToString(f.Reflect().Interface()))
}
