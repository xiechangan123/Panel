package caddy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// TestValidateWithCaddy 用真实的 caddy 二进制校验生成的完整配置，CADDY_BIN 未设置时跳过
func TestValidateWithCaddy(t *testing.T) {
	bin := os.Getenv("CADDY_BIN")
	if bin == "" {
		t.Skip("CADDY_BIN not set")
	}

	root := t.TempDir()
	// 指定目录时布局保留在该目录，便于本机启动 caddy 做冒烟测试
	if smokeRoot := os.Getenv("CADDY_SMOKE_DIR"); smokeRoot != "" {
		require.NoError(t, os.RemoveAll(smokeRoot))
		require.NoError(t, os.MkdirAll(smokeRoot, 0755))
		root = smokeRoot
	}
	sites := filepath.Join(root, "sites")
	certPath, keyPath := writeSelfSigned(t, root)

	newSite := func(name string) string {
		configDir := filepath.Join(sites, name, "config")
		for _, dir := range []string{"site", "shared"} {
			require.NoError(t, os.MkdirAll(filepath.Join(configDir, dir), 0755))
		}
		require.NoError(t, os.MkdirAll(filepath.Join(sites, name, "public"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(sites, name, "log"), 0755))
		return configDir
	}
	d := Dialect{}

	// PHP 站点：SSL、认证、重定向与全部类型片段
	phpDir := newSite("php")
	php, err := NewPHPVhost(phpDir)
	require.NoError(t, err)
	require.NoError(t, php.SetListen([]types.Listen{{Address: "80"}, {Address: "443", Args: []string{"ssl"}}}))
	require.NoError(t, php.SetServerName([]string{"php.example.com", "www.php.example.com"}))
	require.NoError(t, php.SetRoot(filepath.Join(sites, "php", "public")))
	require.NoError(t, php.SetIndex([]string{"index.php", "index.html"}))
	require.NoError(t, php.SetAccessLog(filepath.Join(sites, "php", "log", "access.log")))
	require.NoError(t, php.SetPHP(84))
	require.NoError(t, php.SetSSLConfig(&types.SSLConfig{Cert: certPath, Key: keyPath, Protocols: []string{"TLSv1.2", "TLSv1.3"}, HSTS: true, HTTPRedirect: true}))
	htpasswd := filepath.Join(sites, "php", "htpasswd_0")
	require.NoError(t, os.WriteFile(htpasswd, []byte(d.HTPasswdLine("admin", "secret")+"\n"), 0644))
	require.NoError(t, php.SetBasicAuth([]types.BasicAuth{{Path: "/", UserFile: htpasswd}, {Path: "/admin", UserFile: htpasswd}}))
	require.NoError(t, php.SetRedirects([]types.Redirect{
		{Type: types.RedirectTypeHost, From: "old.example.com", To: "https://php.example.com", KeepURI: true, StatusCode: 301},
		{Type: types.RedirectTypeURL, From: "/old", To: "/new", StatusCode: 302},
		{Type: types.RedirectType404, To: "/", StatusCode: 308},
	}))
	require.NoError(t, php.SetConfig("010-error-404.conf", types.ScopeSite, d.ErrorPageConf()))
	require.NoError(t, php.SetConfig("010-cache.conf", types.ScopeSite, d.PHPCacheConf()))
	require.NoError(t, php.Save())

	// 每个伪静态预置各占一个站点，预置之间的匹配器名可能重复，真实站点一次只用一个
	presets, _ := filepath.Glob(filepath.Join("..", "..", "embed", "rewrites", "caddy", "*.conf"))
	require.NotEmpty(t, presets)
	for _, preset := range presets {
		name := strings.TrimSuffix(filepath.Base(preset), ".conf")
		content, err := os.ReadFile(preset)
		require.NoError(t, err)
		site, err := NewPHPVhost(newSite("preset-" + name))
		require.NoError(t, err)
		require.NoError(t, site.SetServerName([]string{name + ".preset.test"}))
		require.NoError(t, site.SetRoot(filepath.Join(sites, "preset-"+name, "public")))
		require.NoError(t, site.SetPHP(84))
		require.NoError(t, site.SetRawConfig("010-rewrite.conf", types.ScopeSite, string(content)))
		require.NoError(t, site.Save())
	}

	// 反向代理站点：上游片段、各种匹配方式与代理选项
	proxyDir := newSite("proxy")
	proxy, err := NewProxyVhost(proxyDir)
	require.NoError(t, err)
	require.NoError(t, proxy.SetListen([]types.Listen{{Address: "80"}, {Address: "8443", Args: []string{"ssl"}}}))
	require.NoError(t, proxy.SetServerName([]string{"proxy.example.com"}))
	require.NoError(t, proxy.SetRoot(filepath.Join(sites, "proxy", "public")))
	require.NoError(t, proxy.SetSSLConfig(&types.SSLConfig{Cert: certPath, Key: keyPath, Protocols: []string{"TLSv1.3"}}))
	require.NoError(t, proxy.SetUpstreams([]types.Upstream{
		{Name: "backend", Servers: map[string]string{"127.0.0.1:3001": "weight=5", "127.0.0.1:3002": ""}},
		{Name: "pool", Servers: map[string]string{"unix:/tmp/api.sock": "", "10.0.0.2:80": ""}, Algo: "least_conn"},
	}))
	require.NoError(t, proxy.SetProxies([]types.Proxy{
		{Location: "/", Pass: "http://backend", Host: "$proxy_host", Buffering: true, Headers: map[string]string{"X-Real-IP": "$remote_addr"}, Replaces: map[string]string{"http://old": "https://new"}},
		{
			Location: "^~ /api/", Pass: "https://10.0.0.5/v2/", SNI: "api.internal", HTTPVersion: "2",
			Timeout: &types.TimeoutConfig{Connect: 5 * time.Second, Read: 90 * time.Second},
			Retry:   &types.RetryConfig{Tries: 3, Timeout: 10 * time.Second}, ClientMaxBodySize: 10485760,
			SSLBackend:      &types.SSLBackendConfig{Verify: true, TrustedCertificate: certPath},
			ResponseHeaders: &types.ResponseHeaderConfig{Hide: []string{"Server"}, Add: map[string]string{"X-Proxy": "caddy"}},
			AccessControl:   &types.AccessControlConfig{Allow: []string{"10.0.0.0/8"}, Deny: []string{"10.0.0.99"}},
		},
		{Location: "~* \\.(jpg|png)$", Pass: "http://pool", Buffering: true},
		{Location: "= /health", Pass: "http://unix:/tmp/app.sock", Buffering: true},
		{Location: "/ws", Pass: "https://backend.example.com", Buffering: false},
	}))
	require.NoError(t, proxy.SetConfig("010-error-404.conf", types.ScopeSite, d.ErrorPageConf()))
	require.NoError(t, proxy.Save())

	// 停用的静态站点，带 IP 绑定与 SPA 片段
	staticDir := newSite("static")
	static, err := NewStaticVhost(staticDir)
	require.NoError(t, err)
	require.NoError(t, static.SetListen([]types.Listen{{Address: "127.0.0.1:80"}, {Address: "127.0.0.1:443", Args: []string{"ssl"}}}))
	require.NoError(t, static.SetServerName([]string{"static.example.com"}))
	require.NoError(t, static.SetRoot(filepath.Join(sites, "static", "public")))
	require.NoError(t, static.SetSSLConfig(&types.SSLConfig{Cert: certPath, Key: keyPath}))
	require.NoError(t, static.SetRawConfig("800-spa.conf", types.ScopeSite, d.SPAConf()))
	require.NoError(t, static.SetEnable(false))
	require.NoError(t, static.Save())

	// phpMyAdmin 式无域名站点
	pmaDir := newSite("phpmyadmin")
	require.NoError(t, os.WriteFile(filepath.Join(pmaDir, ConfName), []byte(":888 {\n\troot * "+filepath.Join(sites, "phpmyadmin", "public")+"\n\tphp_fastcgi unix//tmp/php-cgi-83.sock\n\tfile_server {\n\t\tindex index.php\n\t}\n}\n"), 0600))

	main := strings.NewReplacer("{root}", root, "{sites}", sites, "{cert}", certPath, "{key}", keyPath).Replace(`{
	admin unix/{root}/admin.sock
	auto_https off
	persist_config off
	grace_period 10s
	default_sni localhost
	fallback_sni localhost
	storage file_system {
		root {root}/data
	}
	log {
		output file {root}/error.log
		exclude http.log.access
	}
}

import {sites}/*/config/caddy.conf

:80 {
	root * {root}/html
	handle /.well-known/acme-challenge/* {
		root * {root}/acme
		file_server
	}
	file_server
}

:443 {
	tls {cert} {key}
	root * {root}/html
	file_server
}
`)
	// 冒烟站点：高位端口，可在本机启动后用 curl 验证行为
	smokeDir := newSite("smoke")
	smoke, err := NewStaticVhost(smokeDir)
	require.NoError(t, err)
	require.NoError(t, smoke.SetListen([]types.Listen{{Address: "18080"}, {Address: "18443", Args: []string{"ssl"}}}))
	require.NoError(t, smoke.SetServerName([]string{"localhost"}))
	require.NoError(t, smoke.SetRoot(filepath.Join(sites, "smoke", "public")))
	require.NoError(t, smoke.SetAccessLog(filepath.Join(sites, "smoke", "log", "access.log")))
	require.NoError(t, smoke.SetSSLConfig(&types.SSLConfig{Cert: certPath, Key: keyPath, HSTS: true, HTTPRedirect: true}))
	require.NoError(t, smoke.SetBasicAuth([]types.BasicAuth{{Path: "/admin", UserFile: htpasswd}}))
	require.NoError(t, smoke.SetRedirects([]types.Redirect{{Type: types.RedirectTypeURL, From: "/old", To: "/new", StatusCode: 302}}))
	require.NoError(t, smoke.SetRawConfig("800-spa.conf", types.ScopeSite, d.SPAConf()))
	require.NoError(t, smoke.SetConfig("010-error-404.conf", types.ScopeSite, d.ErrorPageConf()))
	_, stat := d.StatConf("smoke")
	require.NoError(t, smoke.SetConfig("021-stats-log.conf", types.ScopeSite, stat))
	require.NoError(t, smoke.SetDefault(true))
	require.NoError(t, smoke.Save())
	require.NoError(t, os.WriteFile(filepath.Join(sites, "smoke", "public", "index.html"), []byte("smoke index"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sites, "smoke", "public", "404.html"), []byte("smoke 404"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(sites, "smoke", "public", "admin"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sites, "smoke", "public", "admin", "index.html"), []byte("admin area"), 0644))
	stoppedDir := newSite("stopped")
	stopped, err := NewStaticVhost(stoppedDir)
	require.NoError(t, err)
	require.NoError(t, stopped.SetListen([]types.Listen{{Address: "18081"}}))
	require.NoError(t, stopped.SetServerName([]string{"localhost"}))
	require.NoError(t, stopped.SetRoot(filepath.Join(sites, "stopped", "public")))
	require.NoError(t, stopped.SetEnable(false))
	require.NoError(t, stopped.Save())
	require.NoError(t, os.WriteFile(filepath.Join(sites, "stopped", "public", "index.html"), []byte("must not be served"), 0644))

	mainPath := filepath.Join(root, "Caddyfile")
	require.NoError(t, os.WriteFile(mainPath, []byte(main), 0600))

	out, err := exec.Command(bin, "validate", "--config", mainPath).CombinedOutput()
	require.NoError(t, err, "%s\n---- php ----\n%s\n---- proxy ----\n%s", out, readFile(t, filepath.Join(phpDir, ConfName)), readFile(t, filepath.Join(proxyDir, ConfName)))
}

func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

// writeSelfSigned 生成测试用自签名证书
func writeSelfSigned(t *testing.T, dir string) (string, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost", "php.example.com", "www.php.example.com", "proxy.example.com", "static.example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600))
	return certPath, keyPath
}
