package rule

import (
	"net"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/validator"
	"github.com/spf13/cast"
)

// IPCIDR 验证一个值是否是有效的 IP 或 CIDR
type IPCIDR struct {
	t *gotext.Locale
}

func NewIPCIDR(t *gotext.Locale) *IPCIDR {
	return &IPCIDR{t: t}
}

func (r *IPCIDR) Signature() string { return "ipcidr" }

func (r *IPCIDR) Message() string {
	return r.t.Get("{field} must be a valid IP address or CIDR notation")
}

func (r *IPCIDR) Passes(f *validator.Field) bool {
	if validator.IsEmptyValue(f.Reflect()) {
		return true
	}
	str := cast.ToString(f.Reflect().Interface())
	if net.ParseIP(str) != nil {
		return true
	}
	if _, _, err := net.ParseCIDR(str); err == nil {
		return true
	}
	return false
}
