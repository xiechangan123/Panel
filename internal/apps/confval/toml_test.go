package confval

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TOMLTestSuite struct {
	suite.Suite
}

func TestTOMLTestSuite(t *testing.T) {
	suite.Run(t, new(TOMLTestSuite))
}

func (s *TOMLTestSuite) TestGetDottedKey() {
	conf := "bindPort = 7000\nauth.token = \"12345678\"\n# maxPortsPerClient = 0\n"

	s.Equal("7000", GetTOML(conf, "bindPort"))
	s.Equal("12345678", GetTOML(conf, "auth.token"))
	// 注释掉的项视为未设置
	s.Empty(GetTOML(conf, "maxPortsPerClient"))
	s.Empty(GetTOML(conf, "absent"))
}

func (s *TOMLTestSuite) TestGetSectionKey() {
	conf := "bindPort = 7000\n\n[auth]\nmethod = \"token\"\ntoken = \"abc\"\n"

	s.Equal("token", GetTOML(conf, "auth.method"))
	s.Equal("abc", GetTOML(conf, "auth.token"))
}

func (s *TOMLTestSuite) TestGetIgnoresProxyBlock() {
	conf := "serverAddr = \"1.1.1.1\"\n\n[[proxies]]\nname = \"ssh\"\nserverAddr = \"2.2.2.2\"\n"

	s.Equal("1.1.1.1", GetTOML(conf, "serverAddr"))
}

func (s *TOMLTestSuite) TestSetReplacesInPlace() {
	conf := "# 监听端口\nbindPort = 7000 # 默认\nauth.token = \"old\"\n"

	s.Equal("# 监听端口\nbindPort = 7001 # 默认\nauth.token = \"old\"\n", SetTOML(conf, "bindPort", 7001))
	s.Equal("# 监听端口\nbindPort = 7000 # 默认\nauth.token = \"new\"\n", SetTOML(conf, "auth.token", "new"))
}

func (s *TOMLTestSuite) TestSetReplacesInSection() {
	conf := "bindPort = 7000\n\n[auth]\ntoken = \"old\"\n"

	s.Equal("bindPort = 7000\n\n[auth]\ntoken = \"new\"\n", SetTOML(conf, "auth.token", "new"))
}

func (s *TOMLTestSuite) TestSetInsertsBeforeFirstTable() {
	conf := "bindPort = 7000\n\n[[proxies]]\nname = \"ssh\"\n"

	s.Equal("bindPort = 7000\n\nauth.token = \"abc\"\n[[proxies]]\nname = \"ssh\"\n", SetTOML(conf, "auth.token", "abc"))
}

func (s *TOMLTestSuite) TestSetAppendsWhenNoTable() {
	s.Equal("bindPort = 7000\nauth.token = \"abc\"", SetTOML("bindPort = 7000", "auth.token", "abc"))
}

func (s *TOMLTestSuite) TestSetCommentsOutEmptyValue() {
	s.Equal("# bindPort = 7000", SetTOML("bindPort = 7000", "bindPort", ""))
	s.Equal("# bindPort = 7000", SetTOML("# bindPort = 7000", "bindPort", ""))
	// 键不存在时不应插入空值
	s.Equal("bindPort = 7000", SetTOML("bindPort = 7000", "auth.token", ""))
}

func (s *TOMLTestSuite) TestSetReenablesCommented() {
	s.Equal("maxPortsPerClient = 5", SetTOML("# maxPortsPerClient = 0", "maxPortsPerClient", 5))
}

func (s *TOMLTestSuite) TestSetBool() {
	s.Equal("bindPort = 7000\ntransport.tls.force = true", SetTOML("bindPort = 7000", "transport.tls.force", true))
}

func (s *TOMLTestSuite) TestSetDoesNotTouchProxyBlock() {
	conf := "serverAddr = \"1.1.1.1\"\n\n[[proxies]]\n# 手写的隧道\nname = \"ssh\"\nserverPort = 22\n"

	s.Equal("serverAddr = \"2.2.2.2\"\n\n[[proxies]]\n# 手写的隧道\nname = \"ssh\"\nserverPort = 22\n", SetTOML(conf, "serverAddr", "2.2.2.2"))
	// serverPort 只存在于 proxies 表内，不应被改写，而是插到首个表头前
	s.Equal("serverAddr = \"1.1.1.1\"\n\nserverPort = 7000\n[[proxies]]\n# 手写的隧道\nname = \"ssh\"\nserverPort = 22\n", SetTOML(conf, "serverPort", 7000))
}
