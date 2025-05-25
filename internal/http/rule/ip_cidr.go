package rule

import "net"

// IPCIDR 验证一个值是否是一个有效的 IP 或 CIDR 格式
type IPCIDR struct{}

func NewIPCIDR() *IPCIDR {
	return &IPCIDR{}
}

func (r *IPCIDR) Passes(val any, options ...any) bool {
	if str, ok := val.(string); ok {
		if ip := net.ParseIP(str); ip != nil {
			return true // 是有效的 IP
		}
		if _, _, err := net.ParseCIDR(str); err == nil {
			return true // 是有效的 CIDR
		}
	}
	return false // 既不是 IP 也不是 CIDR
}
