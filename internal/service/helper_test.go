package service

import (
	"net/http/httptest"
	"testing"
)

// 代理头给裸 IP、RemoteAddr 带端口，两种形态都要归一到同一个 IP
func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		header     string
		value      string
		want       string
	}{
		{"remote addr", "1.2.3.4:5678", "", "", "1.2.3.4"},
		{"remote addr ipv6", "[2001:db8::1]:5678", "", "", "2001:db8::1"},
		{"bare ip header", "10.0.0.1:5678", "X-Real-IP", "1.2.3.4", "1.2.3.4"},
		{"bare ipv6 header", "10.0.0.1:5678", "X-Real-IP", "2001:db8::1", "2001:db8::1"},
		{"forwarded chain", "10.0.0.1:5678", "X-Forwarded-For", "1.2.3.4, 10.0.0.1", "1.2.3.4"},
		{"header with port", "10.0.0.1:5678", "X-Real-IP", "1.2.3.4:9999", "1.2.3.4"},
		{"header empty falls back", "1.2.3.4:5678", "X-Real-IP", "", "1.2.3.4"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/user/login", nil)
			r.RemoteAddr = c.remoteAddr
			if c.value != "" {
				r.Header.Set(c.header, c.value)
			}
			if got := clientIP(r, c.header); got != c.want {
				t.Fatalf("clientIP = %q, want %q", got, c.want)
			}
		})
	}
}
