package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/acepanel/panel/pkg/webserver/types"
)

type VhostTestSuite struct {
	suite.Suite
	vhost     *Vhost
	configDir string
}

func TestVhostTestSuite(t *testing.T) {
	suite.Run(t, &VhostTestSuite{})
}

func (s *VhostTestSuite) SetupTest() {
	// 创建临时配置目录
	configDir, err := os.MkdirTemp("", "nginx-test-*")
	s.Require().NoError(err)
	s.configDir = configDir

	// 创建 server.d 目录
	err = os.MkdirAll(filepath.Join(configDir, "server.d"), 0755)
	s.Require().NoError(err)

	vhost, err := NewVhost(configDir)
	s.Require().NoError(err)
	s.Require().NotNil(vhost)
	s.vhost = vhost
}

func (s *VhostTestSuite) TearDownTest() {
	// 清理临时目录
	if s.configDir != "" {
		s.NoError(os.RemoveAll(s.configDir))
	}
}

func (s *VhostTestSuite) TestNewVhost() {
	s.Equal(s.configDir, s.vhost.configDir)
	s.NotNil(s.vhost.parser)
}

func (s *VhostTestSuite) TestEnable() {
	// 默认应该是启用状态（没有 00-disable.conf）
	s.True(s.vhost.Enable())

	// 禁用网站
	s.NoError(s.vhost.SetEnable(false))
	s.False(s.vhost.Enable())

	// 验证禁用文件存在
	disableFile := filepath.Join(s.configDir, "server.d", DisableConfName)
	_, err := os.Stat(disableFile)
	s.NoError(err)

	// 重新启用
	s.NoError(s.vhost.SetEnable(true))
	s.True(s.vhost.Enable())

	// 验证禁用文件已删除
	_, err = os.Stat(disableFile)
	s.True(os.IsNotExist(err))
}

func (s *VhostTestSuite) TestDisableConfigContent() {
	// 禁用网站
	s.NoError(s.vhost.SetEnable(false))

	// 读取禁用配置内容
	disableFile := filepath.Join(s.configDir, "server.d", DisableConfName)
	content, err := os.ReadFile(disableFile)
	s.NoError(err)

	// 验证内容包含 503 返回
	s.Contains(string(content), "503")
	s.Contains(string(content), "return")
}

func (s *VhostTestSuite) TestServerName() {
	names := []string{"example.com", "www.example.com", "api.example.com"}
	s.NoError(s.vhost.SetServerName(names))

	got := s.vhost.ServerName()
	s.Len(got, 3)
	s.Equal("example.com", got[0])
	s.Equal("www.example.com", got[1])
	s.Equal("api.example.com", got[2])
}

func (s *VhostTestSuite) TestServerNameEmpty() {
	s.NoError(s.vhost.SetServerName([]string{}))
}

func (s *VhostTestSuite) TestRoot() {
	root := "/var/www/html"
	s.NoError(s.vhost.SetRoot(root))
	s.Equal(root, s.vhost.Root())
}

func (s *VhostTestSuite) TestIndex() {
	index := []string{"index.html", "index.php", "default.html"}
	s.NoError(s.vhost.SetIndex(index))

	got := s.vhost.Index()
	s.Len(got, 3)
	s.Equal(index, got)
}

func (s *VhostTestSuite) TestIndexEmpty() {
	s.NoError(s.vhost.SetIndex([]string{}))
}

func (s *VhostTestSuite) TestListen() {
	listens := []types.Listen{
		{Address: "80", Protocol: "http"},
		{Address: "443", Protocol: "https"},
	}
	s.NoError(s.vhost.SetListen(listens))

	got := s.vhost.Listen()
	s.Len(got, 2)
}

func (s *VhostTestSuite) TestListenWithHTTP3() {
	listens := []types.Listen{
		{Address: "443", Protocol: "http3"},
	}
	s.NoError(s.vhost.SetListen(listens))

	got := s.vhost.Listen()
	s.Len(got, 1)
	s.Equal("http3", got[0].Protocol)
}

func (s *VhostTestSuite) TestHTTPS() {
	s.False(s.vhost.HTTPS())
	s.Nil(s.vhost.SSLConfig())
}

func (s *VhostTestSuite) TestSetSSLConfig() {
	sslConfig := &types.SSLConfig{
		Cert:      "/etc/ssl/cert.pem",
		Key:       "/etc/ssl/key.pem",
		Protocols: []string{"TLSv1.2", "TLSv1.3"},
		HSTS:      true,
		OCSP:      true,
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	s.True(s.vhost.HTTPS())

	got := s.vhost.SSLConfig()
	s.NotNil(got)
	s.True(got.HSTS)
	s.True(got.OCSP)
}

func (s *VhostTestSuite) TestSetSSLConfigNil() {
	err := s.vhost.SetSSLConfig(nil)
	s.Error(err)
}

func (s *VhostTestSuite) TestClearHTTPS() {
	sslConfig := &types.SSLConfig{
		Cert: "/etc/ssl/cert.pem",
		Key:  "/etc/ssl/key.pem",
		HSTS: true,
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))
	s.True(s.vhost.HTTPS())

	s.NoError(s.vhost.ClearHTTPS())
	s.False(s.vhost.HTTPS())
}

func (s *VhostTestSuite) TestPHP() {
	s.Equal(0, s.vhost.PHP())

	s.NoError(s.vhost.SetPHP(84))

	// Nginx 的 PHP 实现使用 include 文件
	includes := s.vhost.Includes()
	found := false
	for _, inc := range includes {
		if strings.Contains(inc.Path, "enable-php-84.conf") {
			found = true
			break
		}
	}
	s.True(found, "PHP include file should exist")

	s.NoError(s.vhost.SetPHP(0))
}

func (s *VhostTestSuite) TestAccessLog() {
	accessLog := "/var/log/nginx/access.log"
	s.NoError(s.vhost.SetAccessLog(accessLog))
	s.Equal(accessLog, s.vhost.AccessLog())
}

func (s *VhostTestSuite) TestErrorLog() {
	errorLog := "/var/log/nginx/error.log"
	s.NoError(s.vhost.SetErrorLog(errorLog))
	s.Equal(errorLog, s.vhost.ErrorLog())
}

func (s *VhostTestSuite) TestIncludes() {
	includes := []types.IncludeFile{
		{Path: "/etc/nginx/conf.d/ssl.conf"},
		{Path: "/etc/nginx/conf.d/php.conf"},
	}
	s.NoError(s.vhost.SetIncludes(includes))

	got := s.vhost.Includes()
	s.Len(got, 2)
	s.Equal(includes[0].Path, got[0].Path)
	s.Equal(includes[1].Path, got[1].Path)
}

func (s *VhostTestSuite) TestBasicAuth() {
	s.Nil(s.vhost.BasicAuth())

	auth := map[string]string{
		"realm":     "Test Realm",
		"user_file": "/etc/nginx/htpasswd",
	}
	s.NoError(s.vhost.SetBasicAuth(auth))

	got := s.vhost.BasicAuth()
	s.NotNil(got)
	s.Equal(auth["user_file"], got["user_file"])

	s.NoError(s.vhost.SetBasicAuth(nil))
	s.Nil(s.vhost.BasicAuth())
}

func (s *VhostTestSuite) TestRateLimit() {
	s.Nil(s.vhost.RateLimit())

	limit := &types.RateLimit{
		Rate: "512k",
		Options: map[string]string{
			"perip": "10",
		},
	}
	s.NoError(s.vhost.SetRateLimit(limit))

	got := s.vhost.RateLimit()
	s.NotNil(got)
	s.Equal("512k", got.Rate)

	s.NoError(s.vhost.SetRateLimit(nil))
	s.Nil(s.vhost.RateLimit())
}

func (s *VhostTestSuite) TestReset() {
	err := s.vhost.SetServerName([]string{"modified.com"})
	s.NoError(err)
	err = s.vhost.SetRoot("/modified/path")
	s.NoError(err)

	err = s.vhost.Reset()
	s.NoError(err)

	names := s.vhost.ServerName()
	s.NotContains(names, "modified.com")
}

func (s *VhostTestSuite) TestSave() {
	// 设置配置文件路径
	configFile := filepath.Join(s.configDir, "nginx.conf")
	s.vhost.parser.SetConfigPath(configFile)

	s.NoError(s.vhost.SetServerName([]string{"save-test.com"}))

	s.NoError(s.vhost.Save())

	// 验证配置文件已保存
	content, err := os.ReadFile(configFile)
	s.NoError(err)
	s.Contains(string(content), "save-test.com")
}

func (s *VhostTestSuite) TestDump() {
	err := s.vhost.SetServerName([]string{"dump-test.com"})
	s.NoError(err)
	err = s.vhost.SetRoot("/var/www/dump-test")
	s.NoError(err)

	content := s.vhost.parser.Dump()
	s.NotEmpty(content)
	s.Contains(content, "dump-test.com")
	s.Contains(content, "/var/www/dump-test")
	s.Contains(content, "server")
}

func (s *VhostTestSuite) TestDumpWithSSL() {
	sslConfig := &types.SSLConfig{
		Cert:      "/etc/ssl/cert.pem",
		Key:       "/etc/ssl/key.pem",
		Protocols: []string{"TLSv1.2", "TLSv1.3"},
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	content := s.vhost.parser.Dump()
	s.Contains(content, "ssl_certificate")
	s.Contains(content, "ssl_certificate_key")
}

func (s *VhostTestSuite) TestHTTPSRedirect() {
	sslConfig := &types.SSLConfig{
		Cert:         "/etc/ssl/cert.pem",
		Key:          "/etc/ssl/key.pem",
		HTTPRedirect: true,
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	got := s.vhost.SSLConfig()
	s.NotNil(got)
	s.True(got.HTTPRedirect)
}

func (s *VhostTestSuite) TestAltSvc() {
	sslConfig := &types.SSLConfig{
		Cert:   "/etc/ssl/cert.pem",
		Key:    "/etc/ssl/key.pem",
		AltSvc: `h3=":$server_port"; ma=2592000`,
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	got := s.vhost.SSLConfig()
	s.NotNil(got)
	s.Contains(got.AltSvc, "h3=")
}

func (s *VhostTestSuite) TestDefaultConfIncludesServerD() {
	// 验证默认配置包含 server.d 的 include
	s.Contains(DefaultConf, "server.d")
	s.Contains(DefaultConf, "include")
}
