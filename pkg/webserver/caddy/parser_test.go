package caddy

import (
	"strings"
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

func (s *ParserTestSuite) TestStructure() {
	cfg, err := Parse(`# header
{
	admin off
}

import /etc/caddy/sites/*.conf

(snippet) {
	encode gzip
}

http://a.com:80, https://a.com:443,
https://www.a.com:443 {
	root * /var/www # trailing
	@m {
		path /x*
	}
	respond @m "hello world" 200
	tls cert.pem key.pem {
		protocols tls1.2 tls1.3
	}
}
`)
	s.Require().NoError(err)
	s.Equal([]string{"header"}, cfg.Comments())

	global := cfg.All()[0]
	s.NotNil(global.Block)
	s.Equal("", global.Name)
	s.Empty(global.Args)
	s.Equal("off", global.Get("admin").Arg(0))

	s.Equal("/etc/caddy/sites/*.conf", cfg.Get("import").Arg(0))
	s.Equal("(snippet)", cfg.All()[2].Name)
	s.False(isSite(cfg.All()[2]))

	site := sites(cfg)[0]
	s.Require().NotNil(site)
	s.Equal([]string{"http://a.com:80", "https://a.com:443", "https://www.a.com:443"}, addresses(site))
	root := site.Get("root")
	s.Equal([]string{"*", "/var/www"}, root.Values())
	s.Equal(" trailing", root.Trailing)
	s.Equal("/x*", site.Get("@m").Get("path").Arg(0))
	s.Equal([]string{"@m", "hello world", "200"}, site.Get("respond").Values())
	s.Equal([]string{"tls1.2", "tls1.3"}, site.Get("tls").Get("protocols").Values())
}

func (s *ParserTestSuite) TestLexer() {
	cfg, err := Parse("respond `{\"a\": \"b\"}` 200\nheader X-A \"with \\\"quote\\\"\" \\\n  X-B 1\nrespond <<EOT\n\tline1\n\t  line2\n\tEOT 200\n")
	s.Require().NoError(err)
	all := cfg.All()
	s.Require().Len(all, 3)
	s.Equal([]string{`{"a": "b"}`, "200"}, all[0].Values())
	s.Equal([]string{"X-A", `with "quote"`, "X-B", "1"}, all[1].Values())
	s.Equal([]string{"line1\n  line2", "200"}, all[2].Values())

	// token 中间的反斜杠换行同样续行
	cfg, err = Parse("header X-A val\\\n  X-B 1\n")
	s.Require().NoError(err)
	s.Equal([]string{"X-A", "val", "X-B", "1"}, cfg.All()[0].Values())

	_, err = Parse("a {\n")
	s.Error(err)
	_, err = Parse("}\n")
	s.Error(err)
	_, err = Parse("a \"unterminated\n")
	s.Error(err)
}

func (s *ParserTestSuite) TestRender() {
	cfg := &conf.Config{}
	cfg.Append(&conf.Comment{Text: " generated"})
	cfg.Add("import", "/a/*.conf")
	site := addSite(cfg, "http://a.com:80", "https://a.com:443")
	site.Add("root", "*", "/var/www")
	site.Add("header", "X-Note", "two words")
	site.Add("@acme", "expression", `{path}.startsWith("/x")`)
	site.AddBlock("handle", "@acme").Add("file_server")
	site.Append(&conf.Blank{})
	site.Add("respond", "{")
	cfg.AddBlock("(snip)").Add("encode", "gzip")

	expected := "# generated\nimport /a/*.conf\n\nhttp://a.com:80,\nhttps://a.com:443 {\n\troot * /var/www\n\theader X-Note \"two words\"\n\t@acme expression `{path}.startsWith(\"/x\")`\n\thandle @acme {\n\t\tfile_server\n\t}\n\n\trespond \"{\"\n}\n\n(snip) {\n\tencode gzip\n}\n"
	s.Equal(expected, Export(cfg))

	// 解析不保留块内空行
	reparsed, err := Parse(Export(cfg))
	s.Require().NoError(err)
	s.Equal(strings.Replace(expected, "\t}\n\n\trespond", "\t}\n\trespond", 1), Export(reparsed))
	s.Equal([]string{"two words"}, sites(reparsed)[0].Get("header").Values()[1:])
	s.Equal(`{path}.startsWith("/x")`, sites(reparsed)[0].Get("@acme").Arg(1))
}

func (s *ParserTestSuite) TestMeta() {
	cfg, err := Parse("handle @p {\n\t# ace:location ^~ /api\n\t# ace:pass http://backend\n\treverse_proxy 127.0.0.1:3000\n}\n")
	s.Require().NoError(err)
	h := cfg.Get("handle")
	s.Equal("^~ /api", h.Meta("location"))
	s.Equal("http://backend", h.Meta("pass"))
	s.Equal("", h.Meta("missing"))
	s.Equal("", h.Get("missing").Arg(0))
}
