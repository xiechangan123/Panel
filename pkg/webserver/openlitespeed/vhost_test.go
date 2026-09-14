package openlitespeed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

type VhostTestSuite struct {
	suite.Suite
	configDir string
}

func TestVhostTestSuite(t *testing.T) {
	suite.Run(t, &VhostTestSuite{})
}

func (s *VhostTestSuite) SetupTest() {
	siteDir, err := os.MkdirTemp("", "ols-test-*")
	s.Require().NoError(err)
	s.configDir = filepath.Join(siteDir, "config")
	s.Require().NoError(os.MkdirAll(filepath.Join(s.configDir, "site"), 0755))
	s.Require().NoError(os.MkdirAll(filepath.Join(s.configDir, "shared"), 0755))
}

func (s *VhostTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll(filepath.Dir(s.configDir)))
}

func (s *VhostTestSuite) conf() string {
	content, err := os.ReadFile(filepath.Join(s.configDir, VhostConfName))
	s.Require().NoError(err)
	return string(content)
}

func (s *VhostTestSuite) TestDefaults() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.True(vhost.Enable())
	s.Equal(filepath.Join(filepath.Dir(s.configDir), "public"), vhost.Root())
	s.Equal([]string{"index.html"}, vhost.Index())
	s.Empty(vhost.Listen())
	s.False(vhost.SSL())
	s.Nil(vhost.SSLConfig())
	s.Equal(uint(0), vhost.PHP())
}

func (s *VhostTestSuite) TestBasicRoundTrip() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetListen([]types.Listen{{Address: "80"}, {Address: "443", Args: []string{"ssl", "quic"}}}))
	s.NoError(vhost.SetServerName([]string{"example.com", "www.example.com"}))
	s.NoError(vhost.SetRoot("/var/www/html"))
	s.NoError(vhost.SetIndex([]string{"index.php", "index.html"}))
	s.NoError(vhost.SetAccessLog("/var/log/access.log"))
	s.NoError(vhost.SetErrorLog("/var/log/error.log"))
	s.NoError(vhost.SetPHP(84))
	s.NoError(vhost.SetIncludes([]types.IncludeFile{{Path: "/etc/custom.conf"}}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "docRoot                  /var/www/html")
	s.Contains(conf, "indexFiles               index.php, index.html")
	s.Contains(conf, "accesslog /var/log/access.log {")
	s.Contains(conf, "errorlog /var/log/error.log {")
	s.Contains(conf, "include                  "+phpHandlerFile(84))
	s.Contains(conf, "include                  /etc/custom.conf")

	reloaded, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Listen{{Address: "80", Args: []string{}}, {Address: "443", Args: []string{"ssl", "quic"}}}, reloaded.Listen())
	s.Equal([]string{"example.com", "www.example.com"}, reloaded.ServerName())
	s.Equal("/var/www/html", reloaded.Root())
	s.Equal([]string{"index.php", "index.html"}, reloaded.Index())
	s.Equal("/var/log/access.log", reloaded.AccessLog())
	s.Equal("/var/log/error.log", reloaded.ErrorLog())
	s.Equal(uint(84), reloaded.PHP())
	s.Equal([]types.IncludeFile{{Path: "/etc/custom.conf"}}, reloaded.Includes())

	// 注册片段与监听记录
	register, err := os.ReadFile(filepath.Join(s.configDir, RegisterConfName))
	s.Require().NoError(err)
	s.Contains(string(register), "virtualhost "+filepath.Base(filepath.Dir(s.configDir))+" {")
	s.Contains(string(register), "configFile               "+filepath.Join(s.configDir, VhostConfName))
	listen, err := os.ReadFile(filepath.Join(s.configDir, ListenConfName))
	s.Require().NoError(err)
	s.Contains(string(listen), "listen                   443 ssl quic")
	s.Contains(string(listen), "domain                   example.com www.example.com")
}

func (s *VhostTestSuite) TestEnable() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetEnable(false))
	s.False(vhost.Enable())
	s.NoError(vhost.Save())
	conf := s.conf()
	s.Contains(conf, "context "+stopURI+" {")
	s.Contains(conf, "RewriteRule              ^ "+stopURI+"stop.html [L]")

	s.NoError(vhost.SetEnable(true))
	s.NoError(vhost.Save())
	s.NotContains(s.conf(), stopURI)
}

func (s *VhostTestSuite) TestSSL() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetListen([]types.Listen{{Address: "80"}, {Address: "443", Args: []string{"ssl", "quic"}}}))
	s.NoError(vhost.SetSSLConfig(&types.SSLConfig{
		Cert:         "/path/fullchain.pem",
		Key:          "/path/private.key",
		Protocols:    []string{"TLSv1.2", "TLSv1.3"},
		HSTS:         true,
		OCSP:         true,
		HTTPRedirect: true,
	}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "sslProtocol              24")
	s.Contains(conf, "enableQuic               1")
	s.Contains(conf, "enableStapling           1")
	s.Contains(conf, "Strict-Transport-Security")
	s.Contains(conf, "https://%{HTTP_HOST}%{REQUEST_URI} [R=301,L]")

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.True(reloaded.SSL())
	cfg := reloaded.SSLConfig()
	s.Require().NotNil(cfg)
	s.Equal("/path/fullchain.pem", cfg.Cert)
	s.Equal("/path/private.key", cfg.Key)
	s.Equal([]string{"TLSv1.2", "TLSv1.3"}, cfg.Protocols)
	s.True(cfg.HSTS)
	s.True(cfg.OCSP)
	s.True(cfg.HTTPRedirect)

	s.NoError(reloaded.ClearSSL())
	s.NoError(reloaded.Save())
	s.NotContains(s.conf(), "vhssl")
	s.NotContains(s.conf(), "https://%{HTTP_HOST}")
}

func (s *VhostTestSuite) TestProxy() {
	vhost, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetUpstreams([]types.Upstream{{
		Name:      "backend",
		Servers:   map[string]string{"10.0.0.1:8080": "weight=5", "10.0.0.2:8080": ""},
		Algo:      "least_conn",
		Keepalive: 32,
	}}))
	s.NoError(vhost.SetProxies([]types.Proxy{
		{
			Location:  "^~ /",
			Pass:      "http://backend",
			Host:      "example.com",
			Buffering: true,
		},
		{
			Location:        "~ ^/api/v[0-9]+/",
			Pass:            "https://api.example.com",
			Headers:         map[string]string{"X-Custom": "1"},
			ResponseHeaders: &types.ResponseHeaderConfig{Add: map[string]string{"X-Cache": "HIT"}, Hide: []string{"X-Powered-By"}},
			AccessControl:   &types.AccessControlConfig{Allow: []string{"10.0.0.0/8"}, Deny: []string{"*"}},
		},
		{
			Location: "/ws",
			Pass:     "http://127.0.0.1:3000",
		},
	}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "type                     loadbalancer")
	s.Contains(conf, "workers                  proxy::")
	s.Contains(conf, "address                  https://api.example.com:443")
	s.Contains(conf, "context exp:^/api/v[0-9]+/ {")
	s.Contains(conf, "RequestHeader set Host example.com")
	s.Contains(conf, "Header unset X-Powered-By")
	s.Contains(conf, "allow                    10.0.0.0/8")
	s.Contains(conf, "websocket /ws/ {")
	s.Contains(conf, "address                  127.0.0.1:3000")

	reloaded, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	ups := reloaded.Upstreams()
	s.Require().Len(ups, 1)
	s.Equal("backend", ups[0].Name)
	s.Equal("least_conn", ups[0].Algo)
	s.Equal(32, ups[0].Keepalive)
	s.Equal(map[string]string{"10.0.0.1:8080": "weight=5", "10.0.0.2:8080": ""}, ups[0].Servers)

	proxies := reloaded.Proxies()
	s.Require().Len(proxies, 3)
	s.Equal("^~ /", proxies[0].Location)
	s.Equal("http://backend", proxies[0].Pass)
	s.Equal("example.com", proxies[0].Host)
	s.Equal("~ ^/api/v[0-9]+/", proxies[1].Location)
	s.Equal("https://api.example.com", proxies[1].Pass)
	s.Equal(map[string]string{"X-Custom": "1"}, proxies[1].Headers)
	s.Equal(map[string]string{"X-Cache": "HIT"}, proxies[1].ResponseHeaders.Add)
	s.Equal([]string{"X-Powered-By"}, proxies[1].ResponseHeaders.Hide)
	s.Equal([]string{"10.0.0.0/8"}, proxies[1].AccessControl.Allow)
	s.Equal([]string{"*"}, proxies[1].AccessControl.Deny)
	s.Equal("http://127.0.0.1:3000", proxies[2].Pass)
}

func (s *VhostTestSuite) TestRedirects() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	redirects := []types.Redirect{
		{Type: types.RedirectTypeURL, From: "/old", To: "/new", StatusCode: 301},
		{Type: types.RedirectTypeURL, From: "/docs", To: "https://docs.example.com", KeepURI: true, StatusCode: 302},
		{Type: types.RedirectTypeHost, From: "old.example.com", To: "https://example.com", KeepURI: true, StatusCode: 301},
		{Type: types.RedirectType404, To: "/404.html", StatusCode: 308},
	}
	s.NoError(vhost.SetRedirects(redirects))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "context exp:^/old$ {")
	s.Contains(conf, "context exp:^/docs(.*)$ {")
	s.Contains(conf, "location                 https://docs.example.com$1")
	s.Contains(conf, `RewriteCond              %{HTTP_HOST} ^old\.example\.com$ [NC]`)
	s.Contains(conf, "RewriteRule              ^(.*)$ https://example.com$1 [R=301,L]")
	s.Contains(conf, "errorpage 404 {")

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	got := reloaded.Redirects()
	s.Require().Len(got, 4)
	s.Equal(redirects[0], got[0])
	s.Equal(redirects[1], got[1])
	s.Equal(redirects[3], got[2])
	s.Equal(redirects[2], got[3])
}

func (s *VhostTestSuite) TestBasicAuth() {
	vhost, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetProxies([]types.Proxy{{Location: "^~ /", Pass: "http://127.0.0.1:3000"}}))
	auths := []types.BasicAuth{
		{Path: "/", UserFile: "/opt/ace/sites/x/htpasswd_0"},
		{Path: "/admin", UserFile: "/opt/ace/sites/x/htpasswd_1"},
	}
	s.NoError(vhost.SetBasicAuth(auths))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "realm "+vhost.realmName(0)+" {")
	s.Contains(conf, "location                 /opt/ace/sites/x/htpasswd_1")
	s.Contains(conf, "context /admin/ {")
	// 整站认证合并进代理上下文，不再生成静态根上下文
	s.Equal(1, strings.Count(conf, "context / {"))
	s.Contains(conf, "realm                    "+vhost.realmName(0)+"\n")

	reloaded, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.ElementsMatch(auths, reloaded.BasicAuth())

	s.NoError(reloaded.ClearBasicAuth())
	s.NoError(reloaded.Save())
	s.NotContains(s.conf(), "realm")
}

func (s *VhostTestSuite) TestRootContext() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetSSLConfig(&types.SSLConfig{Cert: "/c", Key: "/k", HSTS: true}))
	s.NoError(vhost.SetBasicAuth([]types.BasicAuth{{Path: "/", UserFile: "/opt/ace/sites/x/htpasswd_0"}}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Equal(1, strings.Count(conf, "context / {"))
	s.Contains(conf, "location                 $DOC_ROOT/")
	s.Contains(conf, "autoLoadHtaccess         1")
	s.Contains(conf, "Header set Strict-Transport-Security max-age=31536000")
	s.NotContains(conf, "\nextraHeaders")

	reloaded, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.True(reloaded.SSLConfig().HSTS)
	s.Equal([]types.BasicAuth{{Path: "/", UserFile: "/opt/ace/sites/x/htpasswd_0"}}, reloaded.BasicAuth())
}

func (s *VhostTestSuite) TestFragments() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetRawConfig("010-rewrite.conf", types.ScopeSite, "RewriteCond %{REQUEST_FILENAME} !-f\nRewriteRule ^ /index.php [L]\n"))
	s.NoError(vhost.SetConfig("010-cache.conf", types.ScopeSite, Dialect{}.PHPCacheConf()))
	s.NoError(vhost.SetConfig("001-acme.conf", types.ScopeSite, ""))
	s.NoError(vhost.SetRawConfig("800-custom.conf", types.ScopeShared, "enableIpGeo 1\n"))
	s.NoError(vhost.Save())

	conf := s.conf()
	rewriteIdx := strings.Index(conf, "\nrewrite {")
	s.Require().Positive(rewriteIdx)
	body := conf[rewriteIdx:]
	s.Contains(body, "include                  "+filepath.Join(s.configDir, "site", "010-rewrite.conf"))
	s.NotContains(body, "010-cache.conf")
	head := conf[:rewriteIdx]
	s.Contains(head, "include                  "+filepath.Join(s.configDir, "site", "010-cache.conf"))
	s.Contains(head, "include                  "+filepath.Join(s.configDir, "shared", "800-custom.conf"))
	s.NotContains(head, "001-acme.conf")
	s.Equal("RewriteCond %{REQUEST_FILENAME} !-f\nRewriteRule ^ /index.php [L]", vhost.Config("010-rewrite.conf", types.ScopeSite))
}

func (s *VhostTestSuite) TestReset() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetPHP(83))
	s.NoError(vhost.SetRoot("/tmp/x"))
	s.NoError(vhost.Reset())
	s.Equal(uint(0), vhost.PHP())
	s.Equal(filepath.Join(filepath.Dir(s.configDir), "public"), vhost.Root())
}

func TestLocationToURI(t *testing.T) {
	cases := map[string]string{
		"^~ /":         "/",
		"/":            "/",
		"/api":         "/api/",
		"^~ /static/":  "/static/",
		"~ ^/v1/":      "exp:^/v1/",
		"~* \\.(jpg)$": `exp:(?i)\.(jpg)$`,
		"= /exact":     "/exact",
		"~ ^/a.b/":     "exp:^/a.b/",
	}
	for in, want := range cases {
		if got := locationToURI(in); got != want {
			t.Errorf("locationToURI(%q) = %q, want %q", in, got, want)
		}
	}
	if got := listenerAddress("80"); got != "*:80" {
		t.Errorf("listenerAddress(80) = %q", got)
	}
	if got := listenerAddress("0.0.0.0:8080"); got != "*:8080" {
		t.Errorf("listenerAddress(0.0.0.0:8080) = %q", got)
	}
	if got := listenerName("[::]:443"); got != "ace_ip6_443" {
		t.Errorf("listenerName = %q", got)
	}
}
