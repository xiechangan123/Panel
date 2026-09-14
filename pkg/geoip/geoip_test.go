package geoip

import (
	"io/fs"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

const testDBPath = "ipipfree.ipdb"

func newTestGeoIP(t *testing.T) *GeoIP {
	t.Helper()
	g, err := NewGeoIP(testDBPath)
	must.NoError(t, err)
	t.Cleanup(func() {
		check.NoError(t, g.Close())
	})
	return g
}

func TestNewGeoIP_InvalidPath(t *testing.T) {
	_, err := NewGeoIP("/nonexistent/path.ipdb")
	check.ErrorIs(t, err, fs.ErrNotExist)
}

func TestLookup_KnownIP(t *testing.T) {
	g := newTestGeoIP(t)
	// 具体归属地随库更新变化，只断言解析出了国家
	for _, ip := range []string{"114.114.114.114", "8.8.8.8"} {
		t.Run(ip, func(t *testing.T) {
			r := g.Lookup(ip)
			check.NotEmpty(t, r.Country, check.Msgf("%s -> %+v", ip, r))
		})
	}
}

func TestLookup_PrivateIP(t *testing.T) {
	// 是否有记录取决于库内容，只要求不 panic
	r := newTestGeoIP(t).Lookup("192.168.1.1")
	t.Logf("192.168.1.1 -> %+v", r)
}

func TestLookup_InvalidIP(t *testing.T) {
	// 查不到时返回整个零值结果，而不是只有国家为空
	r := newTestGeoIP(t).Lookup("not-an-ip")
	check.Equal(t, r, GeoResult{})
}

func TestLookup_NilReceiver(t *testing.T) {
	var g *GeoIP
	r := g.Lookup("8.8.8.8")
	check.Equal(t, r, GeoResult{})
}

func TestLookup_IPv6(t *testing.T) {
	r := newTestGeoIP(t).Lookup("2001:4860:4860::8888")
	t.Logf("2001:4860:4860::8888 -> %+v", r)
}
