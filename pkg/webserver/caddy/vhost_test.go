package caddy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

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
	siteDir, err := os.MkdirTemp("", "caddy-test-*")
	s.Require().NoError(err)
	s.configDir = filepath.Join(siteDir, "config")
	s.Require().NoError(os.MkdirAll(filepath.Join(s.configDir, "site"), 0755))
	s.Require().NoError(os.MkdirAll(filepath.Join(s.configDir, "shared"), 0755))
}

func (s *VhostTestSuite) TearDownTest() {
	s.NoError(os.RemoveAll(filepath.Dir(s.configDir)))
}

// snippet 上游片段名，站点名取自临时目录
func (s *VhostTestSuite) snippet(name string) string {
	return "ace_upstream_" + safeName(filepath.Base(filepath.Dir(s.configDir))) + "_" + safeName(name)
}

func (s *VhostTestSuite) conf() string {
	content, err := os.ReadFile(filepath.Join(s.configDir, ConfName))
	s.Require().NoError(err)
	return string(content)
}

func (s *VhostTestSuite) TestDefaults() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.True(vhost.Enable())
	s.Equal(filepath.Join(filepath.Dir(s.configDir), "public"), vhost.Root())
	s.Equal([]string{"index.html"}, vhost.Index())
	s.Equal([]types.Listen{{Address: "80", Args: []string{}}}, vhost.Listen())
	s.Equal([]string{"localhost"}, vhost.ServerName())
	s.Equal(ErrorLogPath, vhost.ErrorLog())
	s.False(vhost.SSL())
	s.Nil(vhost.SSLConfig())
	s.Equal(uint(0), vhost.PHP())
}

func (s *VhostTestSuite) TestBasicRoundTrip() {
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetListen([]types.Listen{{Address: "80"}, {Address: "443", Args: []string{"ssl"}}}))
	s.NoError(vhost.SetServerName([]string{"example.com", "www.example.com"}))
	s.NoError(vhost.SetRoot("/var/www/html"))
	s.NoError(vhost.SetIndex([]string{"index.php", "index.html"}))
	s.NoError(vhost.SetAccessLog("/var/log/access.log"))
	s.NoError(vhost.SetPHP(84))
	s.NoError(vhost.SetIncludes([]types.IncludeFile{{Path: "/etc/custom.conf"}}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "import "+filepath.Join(s.configDir, "shared", "*.conf")+"\n")
	site := "ace_site_" + safeName(filepath.Base(filepath.Dir(s.configDir)))
	s.Contains(conf, "\n("+site+") {\n")
	s.Contains(conf, "\nhttp://example.com:80,\nhttp://www.example.com:80 {\n\timport "+site+"\n}\n")
	s.Contains(conf, "\nhttps://example.com:443,\nhttps://www.example.com:443 {\n\timport "+site+"\n}\n")
	s.Contains(conf, "\troot * /var/www/html\n")
	s.Contains(conf, "\tencode br zstd gzip\n")
	s.Contains(conf, "\t\toutput file /var/log/access.log\n")
	s.Contains(conf, "\timport "+filepath.Join(s.configDir, "site", "*.conf")+"\n")
	s.Contains(conf, "\timport /etc/custom.conf\n")
	s.Contains(conf, "\tphp_fastcgi unix//tmp/php-cgi-84.sock\n")
	s.Contains(conf, "\t\tindex index.php index.html\n")
	s.Contains(conf, "handle "+acmeMatcher+" {")
	s.NotContains(conf, "bind")
	s.NotContains(conf, "tls ")

	reloaded, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Listen{{Address: "80", Args: []string{}}, {Address: "443", Args: []string{"ssl"}}}, reloaded.Listen())
	s.Equal([]string{"example.com", "www.example.com"}, reloaded.ServerName())
	s.Equal("/var/www/html", reloaded.Root())
	s.Equal([]string{"index.php", "index.html"}, reloaded.Index())
	s.Equal("/var/log/access.log", reloaded.AccessLog())
	s.Equal(uint(84), reloaded.PHP())
	s.Equal([]types.IncludeFile{{Path: "/etc/custom.conf"}}, reloaded.Includes())
	s.Nil(reloaded.SSLConfig())

	// 重新保存不改变内容
	s.NoError(reloaded.Save())
	s.Equal(conf, s.conf())
}

func (s *VhostTestSuite) TestBind() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.Error(vhost.SetListen([]types.Listen{{Address: "80"}, {Address: "1.2.3.4:8080"}}))
	s.NoError(vhost.SetListen([]types.Listen{{Address: "1.2.3.4:80"}, {Address: "[::1]:80"}, {Address: "1.2.3.4:8443", Args: []string{"ssl"}}}))
	s.NoError(vhost.SetServerName([]string{"example.com"}))
	s.NoError(vhost.SetSSLConfig(&types.SSLConfig{Cert: "/c.pem", Key: "/k.pem"}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "\nhttp://example.com:80 {\n\tbind 1.2.3.4 ::1\n")
	s.Contains(conf, "\nhttps://example.com:8443 {\n\tbind 1.2.3.4 ::1\n\ttls /c.pem /k.pem\n")

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Listen{
		{Address: "1.2.3.4:80", Args: []string{}},
		{Address: "[::1]:80", Args: []string{}},
		{Address: "1.2.3.4:8443", Args: []string{"ssl"}},
		{Address: "[::1]:8443", Args: []string{"ssl"}},
	}, reloaded.Listen())
}

func (s *VhostTestSuite) TestHostless() {
	// phpMyAdmin 这类无域名站点，安装脚本写的最简配置也能解析并重新生成
	s.Require().NoError(os.WriteFile(filepath.Join(s.configDir, ConfName), []byte(":888 {\n\troot * /opt/pma\n\tphp_fastcgi unix//tmp/php-cgi-83.sock\n\tfile_server\n}\n"), 0600))
	vhost, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Listen{{Address: "888", Args: []string{}}}, vhost.Listen())
	s.Equal([]string{}, vhost.ServerName())
	s.Equal("/opt/pma", vhost.Root())
	s.Equal(uint(83), vhost.PHP())
	s.Equal("", vhost.AccessLog())

	s.NoError(vhost.SetListen([]types.Listen{{Address: "8888"}}))
	s.NoError(vhost.Save())
	s.Contains(s.conf(), "\nhttp://:8888 {\n")
	s.NotContains(s.conf(), "\tlog {")
	reloaded, err := NewPHPVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Listen{{Address: "8888", Args: []string{}}}, reloaded.Listen())
	s.Equal([]string{}, reloaded.ServerName())
}

func (s *VhostTestSuite) TestEnable() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetEnable(false))
	s.False(vhost.Enable())
	s.NoError(vhost.Save())
	conf := s.conf()
	s.Contains(conf, "\t# ace:stop\n\t"+stopMatcher+" expression true\n\thandle "+stopMatcher+" {\n\t\trewrite * /stop.html\n\t\tfile_server {\n\t\t\troot "+HTMLDir+"\n\t\t}\n\t}\n")

	s.NoError(vhost.SetEnable(true))
	s.NoError(vhost.Save())
	s.NotContains(s.conf(), stopMatcher)
}

func (s *VhostTestSuite) TestSSL() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetListen([]types.Listen{{Address: "80"}, {Address: "443", Args: []string{"ssl"}}}))
	s.NoError(vhost.SetSSLConfig(&types.SSLConfig{
		Cert:         "/path/fullchain.pem",
		Key:          "/path/private.key",
		Protocols:    []string{"TLSv1.3"},
		HSTS:         true,
		OCSP:         false,
		HTTPRedirect: true,
	}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "\ttls /path/fullchain.pem /path/private.key {\n\t\tprotocols tls1.3\n\t}\n")
	s.Contains(conf, "\nhttp://localhost:80 {\n\t"+httpMatcher+" not path /.well-known/acme-challenge/*\n\tredir "+httpMatcher+" https://{host}{uri} 301\n\timport ")
	s.Contains(conf, "\theader Strict-Transport-Security max-age=31536000\n\timport ")

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.True(reloaded.SSL())
	// OCSP 装订由 Caddy 对所有证书自动完成
	s.Equal(&types.SSLConfig{
		Cert:         "/path/fullchain.pem",
		Key:          "/path/private.key",
		Protocols:    []string{"TLSv1.3"},
		HSTS:         true,
		OCSP:         true,
		HTTPRedirect: true,
	}, reloaded.SSLConfig())

	// Caddy 不支持 TLS 1.0/1.1，过滤后与默认相同则省略 protocols
	s.NoError(reloaded.SetSSLConfig(&types.SSLConfig{Cert: "/c", Key: "/k", Protocols: []string{"TLSv1.1", "TLSv1.2", "TLSv1.3"}}))
	s.NoError(reloaded.Save())
	s.Contains(s.conf(), "\ttls /c /k\n")
	s.NotContains(s.conf(), "protocols")
	again, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]string{"TLSv1.2", "TLSv1.3"}, again.SSLConfig().Protocols)
	s.False(again.SSLConfig().HTTPRedirect)
	s.False(again.SSLConfig().HSTS)

	s.NoError(again.ClearSSL())
	s.NoError(again.Save())
	s.NotContains(s.conf(), "tls")
}

func (s *VhostTestSuite) TestBasicAuth() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	siteDir := filepath.Dir(s.configDir)
	file0, file1, file2 := filepath.Join(siteDir, "htpasswd_0"), filepath.Join(siteDir, "htpasswd_1"), filepath.Join(siteDir, "htpasswd_2")
	s.Require().NoError(os.WriteFile(file0, []byte("admin:{PLAIN}secret\n# comment\nbob:plain\n"), 0644))
	s.Require().NoError(os.WriteFile(file1, []byte("ops:{PLAIN}pw\n"), 0644))
	s.Require().NoError(os.WriteFile(file2, []byte(""), 0644))
	s.NoError(vhost.SetBasicAuth([]types.BasicAuth{
		{Path: "/", UserFile: file0},
		{Path: "/admin", UserFile: file1},
		{Path: "/empty", UserFile: file2},
	}))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "\t@ace_auth_0 {\n\t\tpath /*\n\t\tnot path /.well-known/acme-challenge/*\n\t}\n\tbasic_auth @ace_auth_0 {\n\t\timport "+file0+".caddy\n\t}\n")
	s.Contains(conf, "\t@ace_auth_1 path /admin*\n\tbasic_auth @ace_auth_1 {\n\t\timport "+file1+".caddy\n\t}\n")
	s.NotContains(conf, "@ace_auth_2")

	// 明文文件转为 bcrypt 行
	hashed, err := os.ReadFile(file0 + ".caddy")
	s.Require().NoError(err)
	lines := strings.Split(strings.TrimSpace(string(hashed)), "\n")
	s.Require().Len(lines, 2)
	s.True(strings.HasPrefix(lines[0], "admin $2a$"))
	s.True(strings.HasPrefix(lines[1], "bob $2a$"))
	s.NoError(bcrypt.CompareHashAndPassword([]byte(strings.TrimPrefix(lines[0], "admin ")), []byte("secret")))

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.BasicAuth{
		{Path: "/", UserFile: file0},
		{Path: "/admin", UserFile: file1},
	}, reloaded.BasicAuth())

	s.NoError(reloaded.ClearBasicAuth())
	s.NoError(reloaded.Save())
	s.NotContains(s.conf(), "basic_auth")
}

func (s *VhostTestSuite) TestRedirects() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	redirects := []types.Redirect{
		{Type: types.RedirectTypeHost, From: "old.example.com", To: "https://example.com", KeepURI: true, StatusCode: 301},
		{Type: types.RedirectTypeURL, From: "/old", To: "/new", StatusCode: 302},
		{Type: types.RedirectType404, To: "/404-page", StatusCode: 308},
	}
	s.NoError(vhost.SetRedirects(redirects))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "\t@ace_redirect_0 host old.example.com\n\tredir @ace_redirect_0 https://example.com{uri} 301\n")
	s.Contains(conf, "\t@ace_redirect_1 path /old\n\tredir @ace_redirect_1 /new 302\n")
	s.Contains(conf, "\thandle_errors 404 {\n\t\tredir /404-page 308\n\t}\n")

	reloaded, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal(redirects, reloaded.Redirects())
}

func (s *VhostTestSuite) TestProxies() {
	vhost, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	upstreams := []types.Upstream{{
		Name:     "backend",
		Servers:  map[string]string{"127.0.0.1:3001": "weight=5", "127.0.0.1:3002": ""},
		Resolver: []string{},
	}, {
		Name:     "api-pool",
		Servers:  map[string]string{"unix:/tmp/api.sock": "", "10.0.0.2:80": "weight=1 max_fails=3"},
		Algo:     "least_conn",
		Resolver: []string{},
	}}
	s.NoError(vhost.SetUpstreams(upstreams))
	proxies := []types.Proxy{
		{
			Location:  "/",
			Pass:      "http://backend",
			Host:      "$proxy_host",
			Buffering: true,
			Resolver:  []string{},
			Headers:   map[string]string{"X-Real-IP": "$remote_addr"},
			Replaces:  map[string]string{"http://old": "https://new"},
		},
		{
			Location:          "^~ /api/",
			Pass:              "https://10.0.0.5/v2/",
			SNI:               "api.internal",
			Buffering:         false,
			Resolver:          []string{},
			Headers:           map[string]string{},
			Replaces:          map[string]string{},
			HTTPVersion:       "2",
			Timeout:           &types.TimeoutConfig{Connect: 5 * time.Second, Read: 90 * time.Second},
			Retry:             &types.RetryConfig{Tries: 3, Timeout: 10 * time.Second},
			ClientMaxBodySize: 10485760,
			SSLBackend:        &types.SSLBackendConfig{Verify: true, TrustedCertificate: "/ca.pem"},
			ResponseHeaders:   &types.ResponseHeaderConfig{Hide: []string{"Server"}, Add: map[string]string{"X-Proxy": "caddy"}},
			AccessControl:     &types.AccessControlConfig{Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.99"}},
		},
		{
			Location:  "~* \\.(jpg|png)$",
			Pass:      "http://api-pool",
			Buffering: true,
			Resolver:  []string{},
			Headers:   map[string]string{},
			Replaces:  map[string]string{},
		},
		{
			Location:  "= /health",
			Pass:      "http://unix:/tmp/app.sock",
			Buffering: true,
			Resolver:  []string{},
			Headers:   map[string]string{},
			Replaces:  map[string]string{},
		},
	}
	s.NoError(vhost.SetProxies(proxies))
	s.NoError(vhost.Save())

	conf := s.conf()
	s.Contains(conf, "("+s.snippet("backend")+") {\n\t# ace:upstream backend\n\tto 127.0.0.1:3001 127.0.0.1:3002\n\tlb_policy weighted_round_robin 5 1\n}\n")
	s.Contains(conf, "("+s.snippet("api-pool")+") {\n\t# ace:upstream api-pool\n\tto 10.0.0.2:80 unix//tmp/api.sock\n\tlb_policy least_conn\n}\n")
	// 精确、^~ 前缀、正则、普通前缀的顺序
	s.Regexp(`(?s)@ace_proxy_3 path /health.*@ace_proxy_1 path /api/\*.*@ace_proxy_2 path_regexp \(\?i\)\\\.\(jpg\|png\)\$.*@ace_proxy_0 path /\*`, conf)
	s.Contains(conf, "\t\thandle @ace_proxy_0 {\n\t\t\t# ace:location /\n\t\t\t# ace:pass http://backend\n\t\t\treplace http://old https://new\n\t\t\treverse_proxy {\n\t\t\t\timport "+s.snippet("backend")+"\n\t\t\t\theader_up Host {upstream_hostport}\n\t\t\t\theader_up X-Real-IP {remote_host}\n\t\t\t}\n\t\t}\n")
	s.Contains(conf, "\t\t\trequest_body {\n\t\t\t\tmax_size 10485760\n\t\t\t}\n")
	s.Contains(conf, "\t\t\t@ace_deny_1 remote_ip 10.0.0.99\n\t\t\trespond @ace_deny_1 403\n\t\t\t@ace_allow_1 not remote_ip 10.0.0.0/8\n\t\t\trespond @ace_allow_1 403\n")
	s.Contains(conf, "\t\t\turi path_regexp ^/api/ /v2/\n")
	s.Contains(conf, "\t\t\treverse_proxy 10.0.0.5:443 {\n\t\t\t\theader_down X-Proxy caddy\n\t\t\t\theader_down -Server\n\t\t\t\tflush_interval -1\n\t\t\t\tlb_retries 3\n\t\t\t\tlb_try_duration 10s\n\t\t\t\ttransport http {\n\t\t\t\t\ttls\n\t\t\t\t\ttls_server_name api.internal\n\t\t\t\t\ttls_trust_pool file /ca.pem\n\t\t\t\t\tversions h2c 2\n\t\t\t\t\tdial_timeout 5s\n\t\t\t\t\tresponse_header_timeout 1m30s\n\t\t\t\t}\n\t\t\t}\n")
	s.Contains(conf, "\t\t\treverse_proxy unix//tmp/app.sock\n")

	reloaded, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.Equal([]types.Upstream{{
		Name:     "backend",
		Servers:  map[string]string{"127.0.0.1:3001": "weight=5", "127.0.0.1:3002": ""},
		Resolver: []string{},
	}, {
		Name:     "api-pool",
		Servers:  map[string]string{"unix:/tmp/api.sock": "", "10.0.0.2:80": ""},
		Algo:     "least_conn",
		Resolver: []string{},
	}}, reloaded.Upstreams())

	got := reloaded.Proxies()
	s.Require().Len(got, 4)
	s.Equal("/", got[0].Location)
	s.Equal("http://backend", got[0].Pass)
	s.Equal("{upstream_hostport}", got[0].Host)
	s.Equal(map[string]string{"X-Real-IP": "{remote_host}"}, got[0].Headers)
	s.Equal(map[string]string{"http://old": "https://new"}, got[0].Replaces)
	s.True(got[0].Buffering)

	s.Equal("^~ /api/", got[1].Location)
	s.Equal("https://10.0.0.5/v2/", got[1].Pass)
	s.Equal("api.internal", got[1].SNI)
	s.False(got[1].Buffering)
	s.Equal("2", got[1].HTTPVersion)
	s.Equal(&types.TimeoutConfig{Connect: 5 * time.Second, Read: 90 * time.Second}, got[1].Timeout)
	s.Equal(&types.RetryConfig{Tries: 3, Timeout: 10 * time.Second}, got[1].Retry)
	s.Equal(int64(10485760), got[1].ClientMaxBodySize)
	s.Equal(&types.SSLBackendConfig{Verify: true, TrustedCertificate: "/ca.pem"}, got[1].SSLBackend)
	s.Equal(&types.ResponseHeaderConfig{Hide: []string{"Server"}, Add: map[string]string{"X-Proxy": "caddy"}}, got[1].ResponseHeaders)
	s.Equal(&types.AccessControlConfig{Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.99"}}, got[1].AccessControl)

	s.Equal("~* \\.(jpg|png)$", got[2].Location)
	s.Equal("http://api-pool", got[2].Pass)
	s.Equal("= /health", got[3].Location)
	s.Equal("http://unix:/tmp/app.sock", got[3].Pass)
	s.Nil(got[3].SSLBackend)

	// 再次保存内容稳定
	s.NoError(reloaded.Save())
	s.Equal(conf, s.conf())

	s.NoError(reloaded.ClearProxies())
	s.NoError(reloaded.ClearUpstreams())
	s.NoError(reloaded.Save())
	s.NotContains(s.conf(), "route")
	s.NotContains(s.conf(), "(ace_upstream_")
}

func (s *VhostTestSuite) TestHTTPSBackendDefaults() {
	// 未配置后端校验时沿用 nginx 的默认行为：不校验证书
	vhost, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetProxies([]types.Proxy{{Location: "/", Pass: "https://backend.example.com", Buffering: true}}))
	s.NoError(vhost.Save())
	s.Contains(s.conf(), "\t\t\treverse_proxy backend.example.com:443 {\n\t\t\t\ttransport http {\n\t\t\t\t\ttls\n\t\t\t\t\ttls_insecure_skip_verify\n\t\t\t\t}\n\t\t\t}\n")
	reloaded, err := NewProxyVhost(s.configDir)
	s.Require().NoError(err)
	s.Nil(reloaded.Proxies()[0].SSLBackend)
	s.Equal("", reloaded.Proxies()[0].SNI)
}

func (s *VhostTestSuite) TestReset() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetRoot("/custom"))
	s.NoError(vhost.SetServerName([]string{"a.com"}))
	s.NoError(vhost.Reset())
	s.Equal(filepath.Join(filepath.Dir(s.configDir), "public"), vhost.Root())
	s.Equal([]string{"localhost"}, vhost.ServerName())
}

func (s *VhostTestSuite) TestConfigFragments() {
	vhost, err := NewStaticVhost(s.configDir)
	s.Require().NoError(err)
	s.NoError(vhost.SetConfig("010-cache.conf", types.ScopeSite, "encode gzip\n"))
	s.True(strings.HasPrefix(vhost.Config("010-cache.conf", types.ScopeSite), "# Auto-generated"))
	s.NoError(vhost.SetRawConfig("010-rewrite.conf", types.ScopeSite, "try_files {path} /index.html\n"))
	s.Equal("try_files {path} /index.html", vhost.Config("010-rewrite.conf", types.ScopeSite))
	s.NoError(vhost.RemoveConfig("010-rewrite.conf", types.ScopeSite))
	s.Equal("", vhost.Config("010-rewrite.conf", types.ScopeSite))
}

func (s *VhostTestSuite) TestDialect() {
	d := Dialect{}
	s.Equal("caddy", d.Service())
	s.Equal(ConfName, d.ConfigFile())
	s.Equal([]string{"ssl"}, d.HTTPSListenArgs())
	s.Equal(types.Features{}, d.Features())
	s.Equal("caddy", d.RewritesDir())
	s.Equal("admin:{PLAIN}secret", d.HTPasswdLine("admin", "secret"))
	s.NoError(d.BeforeReload())
}
