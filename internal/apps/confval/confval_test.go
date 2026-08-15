package confval

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfValTestSuite struct {
	suite.Suite
}

func TestConfValTestSuite(t *testing.T) {
	suite.Run(t, new(ConfValTestSuite))
}

func (s *ConfValTestSuite) TestDirectiveGet() {
	conf := "# comment\nbind 127.0.0.1 ::1\nport 6379\n\n# maxmemory 100mb\n"

	s.Equal("127.0.0.1 ::1", Directive.Get(conf, "bind"))
	s.Equal("6379", Directive.Get(conf, "port"))
	// 注释掉的项不应被读出
	s.Empty(Directive.Get(conf, "maxmemory"))
	s.Empty(Directive.Get(conf, "absent"))
}

func (s *ConfValTestSuite) TestDirectiveSetReplaces() {
	s.Equal("port 6380", Directive.Set("port 6379", "port", "6380"))
}

func (s *ConfValTestSuite) TestDirectiveSetAppendsWhenMissing() {
	s.Equal("port 6379\ntimeout 30", Directive.Set("port 6379", "timeout", "30"))
}

func (s *ConfValTestSuite) TestDirectiveSetCommentsOutEmptyValue() {
	s.Equal("# port 6379", Directive.Set("port 6379", "port", ""))
	// 已注释的保持原样，不重复加注释符
	s.Equal("# port 6379", Directive.Set("# port 6379", "port", ""))
}

func (s *ConfValTestSuite) TestDirectiveSetReenablesCommented() {
	s.Equal("maxmemory 200mb", Directive.Set("# maxmemory 100mb", "maxmemory", "200mb"))
}

func (s *ConfValTestSuite) TestDirectiveSetOnlyKeepsFirstMatch() {
	s.Equal("port 6380", Directive.Set("port 6379\nport 7000", "port", "6380"))
}

func (s *ConfValTestSuite) TestSetStripsNewlinesFromValue() {
	s.Equal("port 63opq79", Directive.Set("port 1", "port", "63\nopq\r79"))
}

func (s *ConfValTestSuite) TestFTPRequiresValueToMatch() {
	// 裸键行不算配置项，避免把开关误判为键值对
	s.Empty(FTP.Get("NoAnonymous\n", "NoAnonymous"))
	s.Equal("yes", FTP.Get("NoAnonymous yes\n", "NoAnonymous"))
}

func (s *ConfValTestSuite) TestPropertiesRoundTrip() {
	conf := "# broker\nnum.network.threads=3\n"

	s.Equal("3", Properties.Get(conf, "num.network.threads"))
	s.Equal("# broker\nnum.network.threads=5\n", Properties.Set(conf, "num.network.threads", "5"))
}

func (s *ConfValTestSuite) TestNginxTerminatorAndIndent() {
	conf := "http {\n    keepalive_timeout  60;\n}"

	s.Equal("60", Nginx.Get(conf, "keepalive_timeout"))
	// 缩进保留，行尾分号补回
	s.Equal("http {\n    keepalive_timeout 75;\n}", Nginx.Set(conf, "keepalive_timeout", "75"))
}

func (s *ConfValTestSuite) TestNginxIgnoresLinesWithoutTerminator() {
	s.Empty(Nginx.Get("http {\n    keepalive_timeout 60\n}", "keepalive_timeout"))
}

func (s *ConfValTestSuite) TestNginxKeepsTrailingComment() {
	conf := "http {\n    keepalive_timeout  60; # 保持连接\n}"

	// 注释在分号之后，读取时不能混进值里
	s.Equal("60", Nginx.Get(conf, "keepalive_timeout"))
	s.Equal("http {\n    keepalive_timeout 75; # 保持连接\n}", Nginx.Set(conf, "keepalive_timeout", "75"))
}

func (s *ConfValTestSuite) TestCommentOutKeepsWholeLine() {
	// 注释掉时整行保留，行尾注释不丢
	s.Equal("# max_connections = 100 # note", Postgres.Set("max_connections = 100 # note", "max_connections", ""))
}

func (s *ConfValTestSuite) TestNginxAppendsWithIndent() {
	s.Equal("a b;\n    c d;", Nginx.Set("a b;", "c", "d"))
}

func (s *ConfValTestSuite) TestPostgresQuotesAndInlineComment() {
	conf := "max_connections = 100 # note\nlisten_addresses = '*'\n"

	s.Equal("100", Postgres.Get(conf, "max_connections"))
	s.Equal("*", Postgres.Get(conf, "listen_addresses"))
	// 只替换值，行尾注释保留
	s.Equal("max_connections = '200' # note\nlisten_addresses = '*'\n", Postgres.Set(conf, "max_connections", "200"))
}

func (s *ConfValTestSuite) TestInlineCommentNotAppliedWhereUnsupported() {
	// redis.conf 不支持行尾注释，密码里的 # 是数据不能当注释切掉
	s.Equal("p@ss#word", Directive.Get("requirepass p@ss#word\n", "requirepass"))
	s.Equal("requirepass a#b", Directive.Set("requirepass p@ss#word", "requirepass", "a#b"))
	// java properties 同理
	s.Equal("a#b", Properties.Get("pass=a#b\n", "pass"))
}

func (s *ConfValTestSuite) TestQuotedCommentCharIsData() {
	s.Equal("a#b", Postgres.Get("password = 'a#b' # real note\n", "password"))
	s.Equal("password = 'x#y' # real note", Postgres.Set("password = 'a#b' # real note", "password", "x#y"))
}

func (s *ConfValTestSuite) TestINIIgnoresSectionHeaders() {
	conf := "[mysqld]\nport = 3306\n; skip-name-resolve = 1\n"

	s.Equal("3306", INI.Get(conf, "port"))
	s.Empty(INI.Get(conf, "mysqld"))
	// 分号注释同样能被重新启用
	s.Equal("[mysqld]\nport = 3306\nskip-name-resolve = 2\n", INI.Set(conf, "skip-name-resolve", "2"))
}

func (s *ConfValTestSuite) TestSectionINIMatchesOnlyTargetSection() {
	conf := "[server]\nhttp_port = 3000\n\n[database]\nhttp_port = 9999\n"

	s.Equal("3000", SectionINI.GetIn(conf, "server", "http_port"))
	s.Equal("9999", SectionINI.GetIn(conf, "database", "http_port"))
	s.Empty(SectionINI.GetIn(conf, "absent", "http_port"))

	got := SectionINI.SetIn(conf, "database", "http_port", "8888")
	s.Equal("3000", SectionINI.GetIn(got, "server", "http_port"))
	s.Equal("8888", SectionINI.GetIn(got, "database", "http_port"))
}

func (s *ConfValTestSuite) TestSectionINIInsertsInsideSection() {
	conf := "[server]\nhttp_port = 3000\n\n[database]\ntype = sqlite3\n"

	got := SectionINI.SetIn(conf, "server", "domain", "example.com")
	s.Equal("example.com", SectionINI.GetIn(got, "server", "domain"))
	// 新项必须落在 server 段内，不能漏进 database 段
	s.Empty(SectionINI.GetIn(got, "database", "domain"))
}

func (s *ConfValTestSuite) TestSectionINICreatesMissingSection() {
	got := SectionINI.SetIn("[server]\nhttp_port = 3000\n", "smtp", "host", "localhost:25")

	s.Equal("localhost:25", SectionINI.GetIn(got, "smtp", "host"))
	s.Equal("3000", SectionINI.GetIn(got, "server", "http_port"))
}

func (s *ConfValTestSuite) TestPHPINICommentsWithSemicolon() {
	s.Equal("; memory_limit = 128M", PHPINI.Set("memory_limit = 128M", "memory_limit", ""))
}

func (s *ConfValTestSuite) TestGetYAMLFlatAndNested() {
	s.Equal("single-node", GetYAML(map[string]any{"discovery.type": "single-node"}, "discovery.type"))
	s.Equal("single-node", GetYAML(map[string]any{"discovery": map[string]any{"type": "single-node"}}, "discovery.type"))
	s.Empty(GetYAML(map[string]any{}, "discovery.type"))
}

func (s *ConfValTestSuite) TestSetYAMLFlattensAndClearsNested() {
	cfg := map[string]any{"discovery": map[string]any{"type": "old"}}
	SetYAML(cfg, "discovery.type", "single-node")

	s.Equal("single-node", cfg["discovery.type"])
	// 嵌套键清空后整个父键一并移除
	_, ok := cfg["discovery"]
	s.False(ok)
}

func (s *ConfValTestSuite) TestSetYAMLIgnoresEmptyValue() {
	cfg := map[string]any{}
	SetYAML(cfg, "discovery.type", "")

	s.Empty(cfg)
}
