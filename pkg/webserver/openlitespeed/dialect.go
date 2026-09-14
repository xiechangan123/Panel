package openlitespeed

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// acmeConf mod_acme 模块配置，服务器级生效。mod_acme 实现无状态 HTTP-01：
// 对任意 /.well-known/acme-challenge/<token> 直接应答 token.thumbprint，站点配置无需参与验证
const acmeConf = PanelConfDir + "/acme.conf"

// acmeProbe 探测运行中 OLS 是否已按当前指纹应答所用的 token
const acmeProbe = "ace-probe"

// acmeLeaseTTL 指纹租约时长，验证出错时 CleanUp 不会被调用，靠过期避免永久阻塞
const acmeLeaseTTL = 2 * time.Minute

// acmeLease mod_acme 一次只认一个账户指纹，验证进行中不允许切换
var acmeLease struct {
	sync.Mutex
	thumb   string
	active  int
	expires time.Time
}

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

func (Dialect) ConfigFile() string {
	return VhostConfName
}

// PanelACMEConf 站点与面板共用 mod_acme 配置
func (Dialect) PanelACMEConf() string {
	return acmeConf
}

func (Dialect) Features() types.Features {
	return types.Features{LSCache: true}
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

// LSCacheConf 站点级启用 LiteSpeed 页面缓存，缓存目录按站点隔离，由 OLS 自行创建
func (Dialect) LSCacheConf(name string) string {
	cfg := &Config{}
	m := cfg.AddBlock("module", "cache")
	m.Add("ls_enabled", "1")
	m.Add("storagePath", ServerRoot+"/cachedata/"+name)
	return cfg.String()
}

func (Dialect) HTPasswdLine(username, password string) string {
	return username + ":" + password
}

// RewritesDir 重写规则与 Apache 语法兼容，复用 Apache 预置
func (Dialect) RewritesDir() string {
	return "apache"
}

// BeforeReload 重载前重建面板托管的主配置片段，覆盖站点删除等未经 Save 的变更
func (Dialect) BeforeReload() error {
	return Sync()
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

// WriteSiteChallenge 验证由 mod_acme 直接应答，写入当前账户指纹；运行中的 OLS 已按该指纹应答时无需重载
func (Dialect) WriteSiteChallenge(_, path, keyAuth string) (bool, error) {
	thumb, err := acquireThumbprint(path, keyAuth)
	if err != nil {
		return false, err
	}
	return !acmeLive(thumb), nil
}

func (Dialect) RemoveSiteChallenge(_, _, _ string) (bool, error) {
	releaseThumbprint()
	return false, nil
}

// WritePanelChallenge 同一订单的 token 属于同一账户，取任意一个占用一次指纹即可
func (Dialect) WritePanelChallenge(_ string, _ []string, tokens map[string]string) (bool, error) {
	for path, keyAuth := range tokens {
		thumb, err := acquireThumbprint(path, keyAuth)
		if err != nil {
			return false, err
		}
		return !acmeLive(thumb), nil
	}
	return false, nil
}

func (Dialect) RemovePanelChallenge(_ string) (bool, error) {
	releaseThumbprint()
	return false, nil
}

// acquireThumbprint 从 keyAuth（token.thumbprint）取出账户指纹，等到允许切换后写入模块配置
func acquireThumbprint(path, keyAuth string) (string, error) {
	thumb, ok := strings.CutPrefix(keyAuth, filepath.Base(path)+".")
	if !ok || thumb == "" {
		return "", fmt.Errorf("invalid key authorization for %s", path)
	}

	l := &acmeLease
	l.Lock()
	defer l.Unlock()
	for l.active > 0 && l.thumb != thumb && time.Now().Before(l.expires) {
		l.Unlock()
		time.Sleep(time.Second)
		l.Lock()
	}

	cfg := &Config{}
	m := cfg.AddBlock("module", "mod_acme")
	m.Add("ls_enabled", "1")
	m.Add("acmeEnable", "1")
	m.Add("acmeThumbPrint", thumb)
	if err := os.WriteFile(acmeConf, []byte(cfg.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write acme config: %w", err)
	}
	if l.thumb != thumb {
		l.thumb, l.active = thumb, 0
	}
	l.active++
	l.expires = time.Now().Add(acmeLeaseTTL)
	return thumb, nil
}

func releaseThumbprint() {
	acmeLease.Lock()
	if acmeLease.active > 0 {
		acmeLease.active--
	}
	acmeLease.Unlock()
}

// acmeLive 探测运行中的 OLS 是否已按该指纹应答挑战，探测不通一律按需要重载处理
func acmeLive(thumb string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://127.0.0.1/.well-known/acme-challenge/" + acmeProbe)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return err == nil && resp.StatusCode == http.StatusOK && string(body) == acmeProbe+"."+thumb
}
