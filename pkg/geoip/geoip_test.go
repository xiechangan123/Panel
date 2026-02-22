package geoip

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

const testDBPath = "ipipfree.ipdb"

type GeoIPSuite struct {
	suite.Suite
	g *GeoIP
}

func (s *GeoIPSuite) SetupSuite() {
	g, err := NewGeoIP(testDBPath)
	s.Require().NoError(err)
	s.Require().NotNil(g)
	s.g = g
}

func (s *GeoIPSuite) TearDownSuite() {
	s.Require().NoError(s.g.Close())
}

func TestGeoIPSuite(t *testing.T) {
	suite.Run(t, new(GeoIPSuite))
}

// ========== NewGeoIP ==========

func (s *GeoIPSuite) TestNewGeoIP_InvalidPath() {
	_, err := NewGeoIP("/nonexistent/path.ipdb")
	s.Error(err)
}

// ========== Lookup ==========

func (s *GeoIPSuite) TestLookup_ChinaIP() {
	r := s.g.Lookup("114.114.114.114")
	s.NotEmpty(r.Country)
	s.T().Logf("114.114.114.114 -> %s(%s) %s %s ISP=%s", r.Country, r.CountryCode, r.Region, r.City, r.ISP)
}

func (s *GeoIPSuite) TestLookup_ForeignIP() {
	r := s.g.Lookup("8.8.8.8")
	s.NotEmpty(r.Country)
	s.T().Logf("8.8.8.8 -> %s(%s) %s %s ISP=%s", r.Country, r.CountryCode, r.Region, r.City, r.ISP)
}

func (s *GeoIPSuite) TestLookup_PrivateIP() {
	// 内网 IP，不应 panic
	r := s.g.Lookup("192.168.1.1")
	s.T().Logf("192.168.1.1 -> %s(%s) %s %s ISP=%s", r.Country, r.CountryCode, r.Region, r.City, r.ISP)
}

func (s *GeoIPSuite) TestLookup_InvalidIP() {
	r := s.g.Lookup("not-an-ip")
	s.Empty(r.Country)
}

func (s *GeoIPSuite) TestLookup_NilReceiver() {
	var g *GeoIP
	r := g.Lookup("8.8.8.8")
	s.Equal(GeoResult{}, r)
}

func (s *GeoIPSuite) TestLookup_IPv6() {
	// IPv6 地址，数据库可能不支持，不应 panic
	r := s.g.Lookup("2001:4860:4860::8888")
	s.T().Logf("2001:4860:4860::8888 -> %s(%s) %s %s ISP=%s", r.Country, r.CountryCode, r.Region, r.City, r.ISP)
}
