package geoip

import (
	"log/slog"

	"github.com/ipipdotnet/ipdb-go"
)

// GeoIP IP 地理位置解析
type GeoIP struct {
	db *ipdb.City
}

// NewGeoIP 加载 .ipdb 文件，返回 GeoIP 实例
func NewGeoIP(path string) (*GeoIP, error) {
	db, err := ipdb.NewCity(path)
	if err != nil {
		return nil, err
	}
	return &GeoIP{db: db}, nil
}

// GeoResult 地理位置查询结果
type GeoResult struct {
	Country  string
	Region   string
	City     string
	District string
}

// Reload 重新加载 IP 数据库文件
func (g *GeoIP) Reload(path string) error {
	return g.db.Reload(path)
}

// Lookup 查询 IP 的地理位置，失败返回空结果
func (g *GeoIP) Lookup(ip string) GeoResult {
	if g == nil || g.db == nil {
		return GeoResult{}
	}

	info, err := g.db.FindInfo(ip, "CN")
	if err != nil {
		slog.Debug("geoip lookup failed", slog.String("ip", ip), slog.Any("err", err))
		return GeoResult{}
	}

	return GeoResult{
		Country:  info.CountryName,
		Region:   info.RegionName,
		City:     info.CityName,
		District: info.DistrictName,
	}
}
