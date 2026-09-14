package openlitespeed

import (
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
)

func TestParse(t *testing.T) {
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
	must.NoError(t, err)
	check.DeepEqual(t, cfg.Comments(), []string{"comment"})
	check.Equal(t, cfg.Value("docRoot"), "/var/www/")
	check.Equal(t, cfg.GetBlock("index").Value("indexFiles"), "index.php, index.html")
	ctx := cfg.GetBlock("context", "/")
	must.NotNil(t, ctx)
	check.Equal(t, ctx.Value("allowBrowse"), "1")
	check.Equal(t, ctx.GetBlock("rewrite").Value("enable"), "1")
	// heredoc 合并为单个参数，引号风格记在 Arg 上
	check.DeepEqual(t, ctx.Get("extraHeaders").Args, []conf.Arg{{
		Value: "RequestHeader set Host example.com\nHeader unset X-Powered-By",
		Quote: conf.QuoteHeredoc,
	}})
}

func TestRoundTrip(t *testing.T) {
	cfg := &conf.Config{}
	cfg.Append(conf.Cmt("generated"))
	cfg.Add("docRoot", "/var/www/")
	cfg.Add("enableGzip")
	b := cfg.AddBlock("context", "/api/")
	b.Add("type", "proxy")
	b.Append(&conf.Directive{Name: "rules", Args: []conf.Arg{{Value: "RewriteRule ^ /index.php [L]", Quote: conf.QuoteHeredoc}}})
	b.AddMeta("pass", "http://127.0.0.1:8080")

	out := Export(cfg)
	check.Contains(t, out, "docRoot                  /var/www/\n")
	check.Contains(t, out, "enableGzip\n")
	check.Contains(t, out, "rules <<<END_rules\nRewriteRule ^ /index.php [L]\nEND_rules\n")

	parsed, err := Parse(out)
	must.NoError(t, err)
	check.DeepEqual(t, parsed.Comments(), []string{"generated"})
	got := parsed.GetBlock("context", "/api/")
	must.NotNil(t, got)
	check.Equal(t, got.Value("type"), "proxy")
	check.DeepEqual(t, got.Get("rules").Args, []conf.Arg{{Value: "RewriteRule ^ /index.php [L]", Quote: conf.QuoteHeredoc}})
	check.Equal(t, got.Meta("pass"), "http://127.0.0.1:8080")
	check.Equal(t, Export(parsed), out)
}
