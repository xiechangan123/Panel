package job

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/geoip"
)

// resolveIPDBPath 根据设置返回 IPDB 文件路径
func resolveIPDBPath(setting biz.SettingRepo) string {
	ipdbType, _ := setting.Get(biz.SettingKeyIPDBType)
	switch ipdbType {
	case "subscribe":
		return filepath.Join(app.Root, "panel/storage/geo.ipdb")
	case "custom":
		path, _ := setting.Get(biz.SettingKeyIPDBPath)
		return path
	default:
		return ""
	}
}

// refreshGeoIP 检查 IPDB 文件变化并热更新 GeoIP 实例
func refreshGeoIP(setting biz.SettingRepo, current *geoip.GeoIP, curPath string, curModTime time.Time, log *slog.Logger) (*geoip.GeoIP, string, time.Time) {
	path := resolveIPDBPath(setting)

	// 禁用模式，释放内存
	if path == "" {
		return nil, "", time.Time{}
	}

	info, err := os.Stat(path)
	if err != nil {
		if current != nil {
			log.Warn("ipdb file inaccessible, releasing GeoIP", slog.String("path", path), slog.Any("err", err))
		}
		return nil, path, time.Time{}
	}

	modTime := info.ModTime()

	// 路径和修改时间都没变，无需更新
	if current != nil && path == curPath && modTime.Equal(curModTime) {
		return current, path, modTime
	}

	// 同路径文件更新，使用 Reload
	if current != nil && path == curPath {
		if err = current.Reload(path); err != nil {
			log.Warn("failed to reload ipdb", slog.String("path", path), slog.Any("err", err))
			return current, path, curModTime
		}
		log.Info("ipdb reloaded", slog.String("path", path))
		return current, path, modTime
	}

	// 路径变化，重新初始化
	g, err := geoip.NewGeoIP(path)
	if err != nil {
		log.Warn("failed to load ipdb", slog.String("path", path), slog.Any("err", err))
		return nil, path, time.Time{}
	}
	log.Info("ipdb loaded", slog.String("path", path))
	return g, path, modTime
}
