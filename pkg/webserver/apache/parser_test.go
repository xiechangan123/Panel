package apache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParserTestSuite struct {
	suite.Suite
}

func TestParserTestSuite(t *testing.T) {
	suite.Run(t, &ParserTestSuite{})
}

func (s *ParserTestSuite) TestParseSimpleDirective() {
	input := "ServerName www.example.com"

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Directives, 1)

	directive := config.Directives[0]
	s.Equal("ServerName", directive.Name)
	s.Equal([]string{"www.example.com"}, directive.Args)
}

func (s *ParserTestSuite) TestParseDirectiveWithMultipleArgs() {
	input := "Listen 192.168.1.100:80"

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Directives, 1)

	directive := config.Directives[0]
	s.Equal("Listen", directive.Name)
	s.Equal([]string{"192.168.1.100:80"}, directive.Args)
}

func (s *ParserTestSuite) TestParseComment() {
	input := "# This is a comment\nServerName www.example.com"

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Comments, 1)
	s.Len(config.Directives, 1)

	comment := config.Comments[0]
	s.Equal("This is a comment", comment.Text)
	s.Equal(1, comment.Line)

	directive := config.Directives[0]
	s.Equal("ServerName", directive.Name)
}

func (s *ParserTestSuite) TestParseVirtualHost() {
	input := `<VirtualHost *:80>
    ServerName www.example.com
    DocumentRoot /var/www/html
</VirtualHost>`

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.VirtualHosts, 1)

	vhost := config.VirtualHosts[0]
	s.Equal("VirtualHost", vhost.Name)
	s.Equal([]string{"*:80"}, vhost.Args)
	s.Len(vhost.Directives, 2)

	serverName := vhost.Directives[0]
	s.Equal("ServerName", serverName.Name)
	s.Equal([]string{"www.example.com"}, serverName.Args)

	docRoot := vhost.Directives[1]
	s.Equal("DocumentRoot", docRoot.Name)
	s.Equal([]string{"/var/www/html"}, docRoot.Args)
}

func (s *ParserTestSuite) TestParseComplexConfig() {
	input := `# Apache 配置示例
ServerRoot /etc/apache2
ServerName www.example.com:80

# SSL 配置
LoadModule ssl_module modules/mod_ssl.so

<VirtualHost *:80>
    ServerName www.example.com
    DocumentRoot /var/www/html
    ErrorLog logs/error.log
    CustomLog logs/access.log common
</VirtualHost>

<VirtualHost *:443>
    ServerName www.example.com
    DocumentRoot /var/www/html
    SSLEngine on
    SSLCertificateFile /path/to/certificate.crt
    SSLCertificateKeyFile /path/to/private.key
</VirtualHost>`

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	// 检查注释
	s.Len(config.Comments, 2)
	s.Equal("Apache 配置示例", config.Comments[0].Text)
	s.Equal("SSL 配置", config.Comments[1].Text)

	// 检查全局指令
	s.Len(config.Directives, 3)
	s.Equal("ServerRoot", config.Directives[0].Name)
	s.Equal("ServerName", config.Directives[1].Name)
	s.Equal("LoadModule", config.Directives[2].Name)

	// 检查虚拟主机
	s.Len(config.VirtualHosts, 2)

	// HTTP 虚拟主机
	httpVhost := config.VirtualHosts[0]
	s.Equal([]string{"*:80"}, httpVhost.Args)
	s.Len(httpVhost.Directives, 4)

	// HTTPS 虚拟主机
	httpsVhost := config.VirtualHosts[1]
	s.Equal([]string{"*:443"}, httpsVhost.Args)
	s.Len(httpsVhost.Directives, 5)

	// 检查 SSL 指令
	sslEngine := httpsVhost.Directives[2]
	s.Equal("SSLEngine", sslEngine.Name)
	s.Equal([]string{"on"}, sslEngine.Args)
}

func (s *ParserTestSuite) TestParseQuotedStrings() {
	input := `ServerName "www.example.com"
CustomLog "/var/log/apache2/access.log" combined`

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Directives, 2)

	serverName := config.Directives[0]
	s.Equal("ServerName", serverName.Name)
	s.Equal([]string{"\"www.example.com\""}, serverName.Args)

	customLog := config.Directives[1]
	s.Equal("CustomLog", customLog.Name)
	s.Equal([]string{"\"/var/log/apache2/access.log\"", "combined"}, customLog.Args)
}

func (s *ParserTestSuite) TestParseEmptyConfig() {
	input := ""

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Directives, 0)
	s.Len(config.VirtualHosts, 0)
	s.Len(config.Comments, 0)
}

func (s *ParserTestSuite) TestParseOnlyComments() {
	input := `# 这是第一个注释
# 这是第二个注释`

	config, err := ParseString(input)
	s.NoError(err)
	s.NotNil(config)

	s.Len(config.Comments, 2)
	s.Len(config.Directives, 0)
	s.Len(config.VirtualHosts, 0)

	s.Equal("这是第一个注释", config.Comments[0].Text)
	s.Equal("这是第二个注释", config.Comments[1].Text)
}

func (s *ParserTestSuite) TestConfigGetMethods() {
	input := `ServerName www.example.com
ServerAdmin admin@example.com
ServerName backup.example.com

<VirtualHost *:80>
    ServerName www.example.com
    DocumentRoot /var/www/html
</VirtualHost>

<VirtualHost *:443>
    ServerName www.example.com
    DocumentRoot /var/www/secure
</VirtualHost>`

	config, err := ParseString(input)
	s.NoError(err)

	// 测试 GetDirective
	serverName := config.GetDirective("ServerName")
	s.NotNil(serverName)
	s.Equal("ServerName", serverName.Name)
	s.Equal([]string{"www.example.com"}, serverName.Args)

	// 测试 GetDirectives
	serverNames := config.GetDirectives("ServerName")
	s.Len(serverNames, 2)

	// 测试 GetVirtualHost
	vhost := config.GetVirtualHost("*:80")
	s.NotNil(vhost)
	s.Equal([]string{"*:80"}, vhost.Args)

	// 测试虚拟主机中的 GetDirective
	vhostServerName := vhost.GetDirective("ServerName")
	s.NotNil(vhostServerName)
	s.Equal([]string{"www.example.com"}, vhostServerName.Args)
}

func (s *ParserTestSuite) TestLexerTokens() {
	input := `# Comment
ServerName www.example.com
<VirtualHost *:80>
    DocumentRoot "/var/www/html"
</VirtualHost>`

	lexer, err := NewLexer(strings.NewReader(input))
	s.NoError(err)

	// 测试第一个 token - 注释
	token := lexer.NextToken()
	s.Equal(COMMENT, token.Type)
	s.Equal("Comment", token.Value)
	s.Equal(1, token.Line)

	// 跳过换行
	token = lexer.NextToken()
	s.Equal(NEWLINE, token.Type)

	// 测试指令
	token = lexer.NextToken()
	s.Equal(DIRECTIVE, token.Type)
	s.Equal("ServerName", token.Value)

	// 测试参数
	token = lexer.NextToken()
	s.Equal(DIRECTIVE, token.Type)
	s.Equal("www.example.com", token.Value)
}
