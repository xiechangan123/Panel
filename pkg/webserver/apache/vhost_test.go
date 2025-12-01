package apache

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
	configDir, err := os.MkdirTemp("", "apache-test-*")
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
	s.NotNil(s.vhost.config)
	s.NotNil(s.vhost.vhost)
}

func (s *VhostTestSuite) TestEnable() {
	// 默认应该是启用状态（没有 00-disable.conf）
	s.True(s.vhost.Enable())

	// 禁用网站
	err := s.vhost.SetEnable(false)
	s.NoError(err)
	s.False(s.vhost.Enable())

	// 验证禁用文件存在
	disableFile := filepath.Join(s.configDir, "server.d", DisableConfName)
	_, err = os.Stat(disableFile)
	s.NoError(err)

	// 重新启用
	err = s.vhost.SetEnable(true)
	s.NoError(err)
	s.True(s.vhost.Enable())

	// 验证禁用文件已删除
	_, err = os.Stat(disableFile)
	s.True(os.IsNotExist(err))
}

func (s *VhostTestSuite) TestDisableConfigContent() {
	// 禁用网站
	err := s.vhost.SetEnable(false)
	s.NoError(err)

	// 读取禁用配置内容
	disableFile := filepath.Join(s.configDir, "server.d", DisableConfName)
	content, err := os.ReadFile(disableFile)
	s.NoError(err)

	// 验证内容包含 503 返回
	s.Contains(string(content), "503")
	s.Contains(string(content), "RewriteRule")
}

func (s *VhostTestSuite) TestServerName() {
	names := []string{"example.com", "www.example.com", "api.example.com"}
	err := s.vhost.SetServerName(names)
	s.NoError(err)

	got := s.vhost.ServerName()
	s.Len(got, 3)
	s.Equal("example.com", got[0])
	s.Equal("www.example.com", got[1])
	s.Equal("api.example.com", got[2])
}

func (s *VhostTestSuite) TestServerNameEmpty() {
	err := s.vhost.SetServerName([]string{})
	s.NoError(err)
}

func (s *VhostTestSuite) TestRoot() {
	root := "/var/www/html"
	err := s.vhost.SetRoot(root)
	s.NoError(err)
	s.Equal(root, s.vhost.Root())
}

func (s *VhostTestSuite) TestIndex() {
	index := []string{"index.html", "index.php", "default.html"}
	err := s.vhost.SetIndex(index)
	s.NoError(err)

	got := s.vhost.Index()
	s.Len(got, 3)
	s.Equal(index, got)
}

func (s *VhostTestSuite) TestIndexEmpty() {
	err := s.vhost.SetIndex([]string{})
	s.NoError(err)
	s.Nil(s.vhost.Index())
}

func (s *VhostTestSuite) TestListen() {
	listens := []types.Listen{
		{Address: "*:80", Protocol: "http"},
		{Address: "*:443", Protocol: "https"},
	}
	err := s.vhost.SetListen(listens)
	s.NoError(err)

	got := s.vhost.Listen()
	s.Len(got, 2)
	s.Equal("*:80", got[0].Address)
	s.Equal("*:443", got[1].Address)
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
	err := s.vhost.SetSSLConfig(sslConfig)
	s.NoError(err)

	s.True(s.vhost.HTTPS())

	got := s.vhost.SSLConfig()
	s.NotNil(got)
	s.Equal(sslConfig.Cert, got.Cert)
	s.Equal(sslConfig.Key, got.Key)
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

	err := s.vhost.ClearHTTPS()
	s.NoError(err)
	s.False(s.vhost.HTTPS())
}

func (s *VhostTestSuite) TestClearHTTPSPreservesOtherHeaders() {
	// 添加一个非 HSTS 的 Header
	s.vhost.vhost.AddDirective("Header", "set", "X-Custom-Header", "value")

	// 设置 SSL 和 HSTS
	sslConfig := &types.SSLConfig{
		Cert: "/etc/ssl/cert.pem",
		Key:  "/etc/ssl/key.pem",
		HSTS: true,
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	// 清除 HTTPS
	err := s.vhost.ClearHTTPS()
	s.NoError(err)

	// 检查自定义 Header 是否保留
	headers := s.vhost.vhost.GetDirectives("Header")
	s.NotEmpty(headers)
	found := false
	for _, h := range headers {
		if len(h.Args) >= 2 && h.Args[1] == "X-Custom-Header" {
			found = true
			break
		}
	}
	s.True(found, "自定义 Header 应该被保留")
}

func (s *VhostTestSuite) TestPHP() {
	s.Equal(0, s.vhost.PHP())

	err := s.vhost.SetPHP(84)
	s.NoError(err)
	s.NotEqual(0, s.vhost.PHP())

	err = s.vhost.SetPHP(0)
	s.NoError(err)
	s.Equal(0, s.vhost.PHP())
}

func (s *VhostTestSuite) TestAccessLog() {
	accessLog := "/var/log/apache/access.log"
	err := s.vhost.SetAccessLog(accessLog)
	s.NoError(err)
	s.Equal(accessLog, s.vhost.AccessLog())
}

func (s *VhostTestSuite) TestErrorLog() {
	errorLog := "/var/log/apache/error.log"
	err := s.vhost.SetErrorLog(errorLog)
	s.NoError(err)
	s.Equal(errorLog, s.vhost.ErrorLog())
}

func (s *VhostTestSuite) TestIncludes() {
	includes := []types.IncludeFile{
		{Path: "/etc/apache/conf.d/ssl.conf"},
		{Path: "/etc/apache/conf.d/php.conf"},
	}
	err := s.vhost.SetIncludes(includes)
	s.NoError(err)

	got := s.vhost.Includes()
	s.Len(got, 2)
	s.Equal(includes[0].Path, got[0].Path)
	s.Equal(includes[1].Path, got[1].Path)
}

func (s *VhostTestSuite) TestBasicAuth() {
	s.Nil(s.vhost.BasicAuth())

	auth := map[string]string{
		"realm":     "Test Realm",
		"user_file": "/etc/htpasswd",
	}
	err := s.vhost.SetBasicAuth(auth)
	s.NoError(err)

	got := s.vhost.BasicAuth()
	s.NotNil(got)
	s.Equal(auth["user_file"], got["user_file"])

	err = s.vhost.SetBasicAuth(nil)
	s.NoError(err)
	s.Nil(s.vhost.BasicAuth())
}

func (s *VhostTestSuite) TestRateLimit() {
	s.Nil(s.vhost.RateLimit())

	limit := &types.RateLimit{
		Rate: "512",
	}
	err := s.vhost.SetRateLimit(limit)
	s.NoError(err)

	got := s.vhost.RateLimit()
	s.NotNil(got)

	err = s.vhost.SetRateLimit(nil)
	s.NoError(err)
	s.Nil(s.vhost.RateLimit())
}

func (s *VhostTestSuite) TestReset() {
	s.NoError(s.vhost.SetServerName([]string{"modified.com"}))
	s.NoError(s.vhost.SetRoot("/modified/path"))

	err := s.vhost.Reset()
	s.NoError(err)

	names := s.vhost.ServerName()
	s.NotContains(names, "modified.com")
}

func (s *VhostTestSuite) TestSave() {
	s.NoError(s.vhost.SetServerName([]string{"save-test.com"}))

	err := s.vhost.Save()
	s.NoError(err)

	// 验证配置文件已保存
	configFile := filepath.Join(s.configDir, "apache.conf")
	content, err := os.ReadFile(configFile)
	s.NoError(err)
	s.Contains(string(content), "save-test.com")
}

func (s *VhostTestSuite) TestExport() {
	s.NoError(s.vhost.SetServerName([]string{"export-test.com"}))
	s.NoError(s.vhost.SetRoot("/var/www/export-test"))

	content := s.vhost.config.Export()
	s.NotEmpty(content)
	s.Contains(content, "export-test.com")
	s.Contains(content, "/var/www/export-test")
	s.Contains(content, "<VirtualHost")
	s.Contains(content, "</VirtualHost>")
}

func (s *VhostTestSuite) TestExportWithSSL() {
	sslConfig := &types.SSLConfig{
		Cert:      "/etc/ssl/cert.pem",
		Key:       "/etc/ssl/key.pem",
		Protocols: []string{"TLSv1.2", "TLSv1.3"},
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	content := s.vhost.config.Export()
	s.Contains(content, "SSLEngine on")
	s.Contains(content, "SSLCertificateFile")
	s.Contains(content, "SSLCertificateKeyFile")
}

func (s *VhostTestSuite) TestListenProtocolDetection() {
	listens := []types.Listen{
		{Address: "*:443", Protocol: "https"},
	}
	s.NoError(s.vhost.SetListen(listens))

	sslConfig := &types.SSLConfig{
		Cert: "/etc/ssl/cert.pem",
		Key:  "/etc/ssl/key.pem",
	}
	s.NoError(s.vhost.SetSSLConfig(sslConfig))

	got := s.vhost.Listen()
	s.Len(got, 1)
	s.Equal("https", got[0].Protocol)
}

func (s *VhostTestSuite) TestDirectoryBlock() {
	root := "/var/www/test-dir"
	err := s.vhost.SetRoot(root)
	s.NoError(err)

	content := s.vhost.config.Export()
	s.Contains(content, "<Directory "+root+">")
	s.Contains(content, "</Directory>")
}

func (s *VhostTestSuite) TestPHPFilesMatchBlock() {
	err := s.vhost.SetPHP(84)
	s.NoError(err)

	content := s.vhost.config.Export()
	s.Contains(content, "<FilesMatch")
	s.Contains(content, "SetHandler")
	s.True(strings.Contains(content, "php8.4") || strings.Contains(content, "fcgi"))
}

func (s *VhostTestSuite) TestDefaultVhostConfIncludesServerD() {
	// 验证默认配置包含 server.d 的 include
	s.Contains(DefaultVhostConf, "server.d")
	s.Contains(DefaultVhostConf, "IncludeOptional")
}
