package openlitespeed

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParserTestSuite struct {
	suite.Suite
}

func TestParserTestSuite(t *testing.T) {
	suite.Run(t, &ParserTestSuite{})
}

func (s *ParserTestSuite) TestParseBasic() {
	cfg, err := Parse(`# comment
docRoot   /var/www
index {
  useServer 0
  indexFiles index.php, index.html
}
context /api/ {
  type proxy
  extraHeaders <<<END_extraHeaders
RequestHeader set Host example.com
Header unset X-Powered-By
END_extraHeaders
}
`)
	s.Require().NoError(err)
	s.Equal("/var/www", cfg.Value("docRoot"))
	s.Equal([]string{"comment"}, cfg.Comments())

	idx := cfg.Block("index")
	s.Require().NotNil(idx)
	s.Equal("0", idx.Value("useServer"))
	s.Equal([]string{"index.php", "index.html"}, splitList(idx.Value("indexFiles")))

	ctx := cfg.Block("context", "/api/")
	s.Require().NotNil(ctx)
	s.Equal("proxy", ctx.Value("type"))
	headers := ctx.Directive("extraHeaders")
	s.Require().NotNil(headers)
	s.True(headers.Multiline)
	s.Equal("RequestHeader set Host example.com\nHeader unset X-Powered-By", headers.Value)
}

func (s *ParserTestSuite) TestRoundTrip() {
	cfg := &Config{}
	cfg.Add("docRoot", "/var/www")
	b := cfg.AddBlock("errorlog", "/var/log/err.log")
	b.Add("useServer", "0")
	b.AddMeta("note", "hello world")
	b.Append(&Directive{Name: "rules", Value: "RewriteRule ^ /index.php [L]", Multiline: true})

	parsed, err := Parse(cfg.String())
	s.Require().NoError(err)
	s.Equal("/var/www", parsed.Value("docRoot"))
	got := parsed.Block("errorlog")
	s.Require().NotNil(got)
	s.Equal("/var/log/err.log", got.Arg)
	s.Equal("hello world", got.Meta("note"))
	s.Equal("RewriteRule ^ /index.php [L]", got.Value("rules"))
	s.True(got.Directive("rules").Multiline)
}

func (s *ParserTestSuite) TestUnbalanced() {
	_, err := Parse("context / {\n type proxy\n")
	s.Error(err)
	_, err = Parse("}\n")
	s.Error(err)
}

func (s *ParserTestSuite) TestSetRemove() {
	cfg, err := Parse("a 1\nb 2\nb 3\n")
	s.Require().NoError(err)
	cfg.Set("a", "10")
	s.Equal("10", cfg.Value("a"))
	s.Len(cfg.Directives("b"), 2)
	cfg.Remove("b")
	s.Empty(cfg.Directives("b"))
	s.Equal("A", "A")
}
