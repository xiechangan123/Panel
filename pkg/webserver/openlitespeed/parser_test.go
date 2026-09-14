package openlitespeed

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
)

type ParserTestSuite struct {
	suite.Suite
}

func TestParserTestSuite(t *testing.T) {
	suite.Run(t, &ParserTestSuite{})
}

func (s *ParserTestSuite) TestParse() {
	src := `# comment
docRoot                  /var/www/
index {
  indexFiles             index.php, index.html
}
context / {
  location               $DOC_ROOT/
  allowBrowse            1
  extraHeaders           <<<END_extraHeaders
RequestHeader set Host example.com
Header unset X-Powered-By
END_extraHeaders
  rewrite {
    enable               1
  }
}
`
	cfg, err := Parse(src)
	s.Require().NoError(err)
	s.Equal([]string{"comment"}, cfg.Comments())
	s.Equal("/var/www/", cfg.Value("docRoot"))
	s.Equal("index.php, index.html", cfg.GetBlock("index").Value("indexFiles"))
	ctx := cfg.GetBlock("context", "/")
	s.Require().NotNil(ctx)
	s.Equal("1", ctx.Value("allowBrowse"))
	s.Equal("1", ctx.GetBlock("rewrite").Value("enable"))
	headers := ctx.Get("extraHeaders")
	s.Equal(conf.QuoteHeredoc, headers.Args[0].Quote)
	s.Equal("RequestHeader set Host example.com\nHeader unset X-Powered-By", headers.Arg(0))
}

func (s *ParserTestSuite) TestRoundTrip() {
	cfg := &conf.Config{}
	cfg.Append(conf.Cmt("generated"))
	cfg.Add("docRoot", "/var/www/")
	cfg.Add("enableGzip")
	b := cfg.AddBlock("context", "/api/")
	b.Add("type", "proxy")
	b.Append(&conf.Directive{Name: "rules", Args: []conf.Arg{{Value: "RewriteRule ^ /index.php [L]", Quote: conf.QuoteHeredoc}}})
	b.AddMeta("pass", "http://127.0.0.1:8080")

	out := Export(cfg)
	s.Contains(out, "docRoot                  /var/www/\n")
	s.Contains(out, "enableGzip\n")
	s.Contains(out, "rules <<<END_rules\nRewriteRule ^ /index.php [L]\nEND_rules\n")

	parsed, err := Parse(out)
	s.Require().NoError(err)
	s.Equal([]string{"generated"}, parsed.Comments())
	got := parsed.GetBlock("context", "/api/")
	s.Require().NotNil(got)
	s.Equal("proxy", got.Value("type"))
	s.Equal(conf.QuoteHeredoc, got.Get("rules").Args[0].Quote)
	s.Equal("http://127.0.0.1:8080", got.Meta("pass"))
	s.Equal(out, Export(parsed))
}
