package openlitespeed

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// ACMEDir HTTP-01 验证 token 目录，所有站点与面板默认站点的验证上下文均指向此处
const ACMEDir = ServerRoot + "/acme"

// lsapiDir 记录已切换为 LSAPI 协议的 PHP 版本
const lsapiDir = PanelConfDir + "/lsapi"

// Dialect OpenLiteSpeed 方言
type Dialect struct{}

func (Dialect) Service() string {
	return "openlitespeed"
}

func (Dialect) ConfigTest() string {
	return ServerRoot + "/bin/openlitespeed -t 2>&1"
}

func (Dialect) HTMLDir() string {
	return HTMLDir
}

// PanelACMEConf 面板验证的 token 文件名记录，用于清理
func (Dialect) PanelACMEConf() string {
	return PanelConfDir + "/acme-tokens"
}

func (Dialect) Features() types.Features {
	return types.Features{}
}

func (Dialect) HTTPSListenArgs() []string {
	return []string{"ssl", "quic"}
}

func (Dialect) ErrorPageConf() string {
	return errorPageConf
}

func (Dialect) PHPCacheConf() string {
	return phpCacheConf
}

func (Dialect) SPAConf() string {
	return spaConf
}

func (Dialect) HTPasswdLine(username, password string) string {
	return username + ":" + password
}

// RewritesDir 重写规则与 Apache 语法兼容，复用 Apache 预置
func (Dialect) RewritesDir() string {
	return "apache"
}

// BeforeReload 重载前重建监听器配置，覆盖站点删除等未经 Save 的变更
func (Dialect) BeforeReload() error {
	return SyncListeners()
}

func (Dialect) NewStaticVhost(configDir string) (types.StaticVhost, error) {
	vhost, err := NewStaticVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

func (Dialect) NewPHPVhost(configDir string) (types.PHPVhost, error) {
	vhost, err := NewPHPVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

func (Dialect) NewProxyVhost(configDir string) (types.ProxyVhost, error) {
	vhost, err := NewProxyVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

// WriteSiteChallenge 站点验证上下文为静态目录，只需落盘 token 文件
func (Dialect) WriteSiteChallenge(_, path, token string) error {
	return writeToken(path, token)
}

func (Dialect) RemoveSiteChallenge(_, path, _ string) error {
	return removeToken(path)
}

// WritePanelChallenge 面板域名可能落在任意站点或默认站点，token 目录共用，另记录文件名供清理
func (Dialect) WritePanelChallenge(conf string, _ []string, tokens map[string]string) error {
	var names []string
	for path, token := range tokens {
		if err := writeToken(path, token); err != nil {
			return err
		}
		names = append(names, filepath.Base(path))
	}
	return os.WriteFile(conf, []byte(joinLines(names)), 0600)
}

func (Dialect) RemovePanelChallenge(conf string) error {
	raw, err := os.ReadFile(conf)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, name := range splitLines(string(raw)) {
		_ = removeToken(name)
	}
	return os.WriteFile(conf, []byte(""), 0600)
}

func writeToken(path, token string) error {
	if err := os.MkdirAll(ACMEDir, 0755); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ACMEDir, filepath.Base(path)), []byte(token), 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

func removeToken(path string) error {
	if err := os.Remove(filepath.Join(ACMEDir, filepath.Base(path))); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ========== PHP 协议 ==========

// LSPHPPath 面板 PHP 附带的 lsphp 二进制路径
func LSPHPPath(version uint) string {
	return fmt.Sprintf("/opt/ace/server/php/%d/bin/lsphp", version)
}

// LSAPIEnabled 该 PHP 版本是否以 LSAPI 协议运行
func LSAPIEnabled(version uint) bool {
	_, err := os.Stat(filepath.Join(lsapiDir, strconv.FormatUint(uint64(version), 10)))
	return err == nil
}

// SetLSAPI 切换 PHP 版本的运行协议并重写所有使用该版本的站点配置
func SetLSAPI(version uint, enabled bool) error {
	marker := filepath.Join(lsapiDir, strconv.FormatUint(uint64(version), 10))
	if enabled {
		if _, err := os.Stat(LSPHPPath(version)); err != nil {
			return errors.New("lsphp binary not found")
		}
		if err := os.MkdirAll(lsapiDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(marker, []byte(""), 0600); err != nil {
			return err
		}
	} else if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return err
	}

	matches, _ := filepath.Glob(filepath.Join(SitesPath, "*", "config", VhostConfName))
	for _, path := range matches {
		vhost, err := NewPHPVhost(filepath.Dir(path))
		if err != nil || vhost.PHP() != version {
			continue
		}
		if err = vhost.Save(); err != nil {
			return err
		}
	}

	return nil
}

func joinLines(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
}

func splitLines(content string) []string {
	var out []string
	for line := range strings.SplitSeq(content, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
