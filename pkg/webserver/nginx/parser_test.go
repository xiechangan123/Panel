package nginx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NginxTestSuite struct {
	suite.Suite
}

func TestNginxTestSuite(t *testing.T) {
	suite.Run(t, &NginxTestSuite{})
}

func (s *NginxTestSuite) TestRoot() {
	parser, err := NewParser()
	s.NoError(err)
	root, err := parser.GetRoot()
	s.NoError(err)
	s.Equal("/opt/ace/sites/default/public", root)
	s.NoError(parser.SetRoot("/www/wwwroot/test"))
	root, err = parser.GetRoot()
	s.NoError(err)
	s.Equal("/www/wwwroot/test", root)
}

func (s *NginxTestSuite) TestRootWithComment() {
	parser, err := NewParser()
	s.NoError(err)
	root, comment, err := parser.GetRootWithComment()
	s.NoError(err)
	s.Equal("/opt/ace/sites/default/public", root)
	s.Equal([]string(nil), comment)
	s.NoError(parser.SetRootWithComment("/www/wwwroot/test", []string{"# 测试"}))
	root, comment, err = parser.GetRootWithComment()
	s.NoError(err)
	s.Equal("/www/wwwroot/test", root)
	s.Equal([]string{"# 测试"}, comment)
}

func (s *NginxTestSuite) TestIncludes() {
	parser, err := NewParser()
	s.NoError(err)
	includes, comments, err := parser.GetIncludes()
	s.NoError(err)
	s.Equal([]string{"/opt/ace/sites/default/config/vhost/*.conf"}, includes)
	s.Equal([][]string{{"# custom configs"}}, comments)
	s.NoError(parser.SetIncludes([]string{"/www/server/vhost/rewrite/default.conf"}, nil))
	includes, comments, err = parser.GetIncludes()
	s.NoError(err)
	s.Equal([]string{"/www/server/vhost/rewrite/default.conf"}, includes)
	s.Equal([][]string{[]string(nil)}, comments)
	s.NoError(parser.SetIncludes([]string{"/www/server/vhost/rewrite/test.conf"}, [][]string{{"# 伪静态规则测试"}}))
	includes, comments, err = parser.GetIncludes()
	s.NoError(err)
	s.Equal([]string{"/www/server/vhost/rewrite/test.conf"}, includes)
	s.Equal([][]string{{"# 伪静态规则测试"}}, comments)
}

func (s *NginxTestSuite) TestHTTP() {
	parser, err := NewParser()
	s.NoError(err)
	expect, err := os.ReadFile("testdata/http.conf")
	s.NoError(err)
	s.Equal(string(expect), parser.Dump())
}

func (s *NginxTestSuite) TestHTTPSProtocols() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.Equal([]string{"TLSv1.2", "TLSv1.3"}, parser.GetHTTPSProtocols())
	s.NoError(parser.SetHTTPSProtocols([]string{"TLSv1.3"}))
	s.Equal([]string{"TLSv1.3"}, parser.GetHTTPSProtocols())
}

func (s *NginxTestSuite) TestHTTPSCiphers() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.Equal("ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305", parser.GetHTTPSCiphers())
	s.NoError(parser.SetHTTPSCiphers("TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384"))
	s.Equal("TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384", parser.GetHTTPSCiphers())
}

func (s *NginxTestSuite) TestOCSP() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.False(parser.GetOCSP())
	s.NoError(parser.SetOCSP(false))
	s.False(parser.GetOCSP())
	s.NoError(parser.SetOCSP(true))
	s.True(parser.GetOCSP())
	s.NoError(parser.SetOCSP(false))
	s.False(parser.GetOCSP())
}

func (s *NginxTestSuite) TestHSTS() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.False(parser.GetHSTS())
	s.NoError(parser.SetHSTS(false))
	s.False(parser.GetHSTS())
	s.NoError(parser.SetHSTS(true))
	s.True(parser.GetHSTS())
	s.NoError(parser.SetHSTS(false))
	s.False(parser.GetHSTS())
}

func (s *NginxTestSuite) TestHTTPSRedirect() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.False(parser.GetHTTPSRedirect())
	s.NoError(parser.SetHTTPSRedirect(false))
	s.False(parser.GetHTTPSRedirect())
	s.NoError(parser.SetHTTPSRedirect(true))
	s.True(parser.GetHTTPSRedirect())
	s.NoError(parser.SetHTTPSRedirect(false))
	s.False(parser.GetHTTPSRedirect())
}

func (s *NginxTestSuite) TestAltSvc() {
	parser, err := NewParser()
	s.NoError(err)
	s.NoError(parser.SetHTTPSCert("/www/server/vhost/cert/default.pem", "/www/server/vhost/cert/default.key"))
	s.Equal("", parser.GetAltSvc())
	s.NoError(parser.SetAltSvc(`'h3=":$server_port"; ma=2592000'`))
	s.Equal(`'h3=":$server_port"; ma=2592000'`, parser.GetAltSvc())
	s.NoError(parser.SetAltSvc(""))
	s.Equal("", parser.GetAltSvc())
}

func (s *NginxTestSuite) TestAccessLog() {
	parser, err := NewParser()
	s.NoError(err)
	log, err := parser.GetAccessLog()
	s.NoError(err)
	s.Equal("/opt/ace/sites/default/log/access.log", log)
	s.NoError(parser.SetAccessLog("/www/wwwlogs/access.log"))
	log, err = parser.GetAccessLog()
	s.NoError(err)
	s.Equal("/www/wwwlogs/access.log", log)
}

func (s *NginxTestSuite) TestErrorLog() {
	parser, err := NewParser()
	s.NoError(err)
	log, err := parser.GetErrorLog()
	s.NoError(err)
	s.Equal("/opt/ace/sites/default/log/error.log", log)
	s.NoError(parser.SetErrorLog("/www/wwwlogs/error.log"))
	log, err = parser.GetErrorLog()
	s.NoError(err)
	s.Equal("/www/wwwlogs/error.log", log)
}
