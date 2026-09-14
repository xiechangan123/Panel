package caddy

import (
	"strings"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
)

func TestParserStructure(t *testing.T) {
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
	must.NoError(t, err)
	check.DeepEqual(t, cfg.Comments(), []string{"header"})

	all := cfg.All()
	must.Len(t, all, 4)

	global := all[0]
	must.NotNil(t, global.Block)
	check.Equal(t, global.Name, "")
	check.Empty(t, global.Args)
	check.Equal(t, global.Get("admin").Arg(0), "off")

	check.Equal(t, cfg.Get("import").Arg(0), "/etc/caddy/sites/*.conf")
	check.Equal(t, all[2].Name, "(snippet)")
	check.False(t, isSite(all[2]), check.Msgf("%s 不应识别为站点", all[2].Name))

	siteList := sites(cfg)
	must.Len(t, siteList, 1)
	site := siteList[0]
	check.DeepEqual(t, addresses(site), []string{"http://a.com:80", "https://a.com:443", "https://www.a.com:443"})
	root := site.Get("root")
	check.DeepEqual(t, root.Values(), []string{"*", "/var/www"})
	check.Equal(t, root.Trailing, " trailing")
	check.Equal(t, site.Get("@m").Get("path").Arg(0), "/x*")
	check.DeepEqual(t, site.Get("respond").Values(), []string{"@m", "hello world", "200"})
	check.DeepEqual(t, site.Get("tls").Get("protocols").Values(), []string{"tls1.2", "tls1.3"})
}

func TestParserLexer(t *testing.T) {
	cfg, err := Parse("respond `{\"a\": \"b\"}` 200\nheader X-A \"with \\\"quote\\\"\" \\\n  X-B 1\nrespond <<EOT\n\tline1\n\t  line2\n\tEOT 200\n")
	must.NoError(t, err)
	all := cfg.All()
	must.Len(t, all, 3)
	check.DeepEqual(t, all[0].Values(), []string{`{"a": "b"}`, "200"})
	check.DeepEqual(t, all[1].Values(), []string{"X-A", `with "quote"`, "X-B", "1"})
	check.DeepEqual(t, all[2].Values(), []string{"line1\n  line2", "200"})

	// token 中间的反斜杠换行同样续行，续行后仍是一条指令
	cfg, err = Parse("header X-A val\\\n  X-B 1\n")
	must.NoError(t, err)
	all = cfg.All()
	must.Len(t, all, 1)
	check.DeepEqual(t, all[0].Values(), []string{"X-A", "val", "X-B", "1"})
}

func TestParserLexerError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"块未闭合", "a {\n"},
		{"多余的右花括号", "}\n"},
		{"引号未闭合", "a \"unterminated\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			check.Error(t, err)
		})
	}
}

func TestParserRender(t *testing.T) {
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
	check.Equal(t, Export(cfg), expected)

	// 解析不保留块内空行
	reparsed, err := Parse(Export(cfg))
	must.NoError(t, err)
	check.Equal(t, Export(reparsed), strings.Replace(expected, "\t}\n\n\trespond", "\t}\n\trespond", 1))
	reparsedSites := sites(reparsed)
	must.Len(t, reparsedSites, 1)
	check.DeepEqual(t, reparsedSites[0].Get("header").Values(), []string{"X-Note", "two words"})
	check.Equal(t, reparsedSites[0].Get("@acme").Arg(1), `{path}.startsWith("/x")`)
}

func TestParserMeta(t *testing.T) {
	cfg, err := Parse("handle @p {\n\t# ace:location ^~ /api\n\t# ace:pass http://backend\n\treverse_proxy 127.0.0.1:3000\n}\n")
	must.NoError(t, err)
	h := cfg.Get("handle")
	check.Equal(t, h.Meta("location"), "^~ /api")
	check.Equal(t, h.Meta("pass"), "http://backend")
	check.Equal(t, h.Meta("missing"), "")
	check.Equal(t, h.Get("missing").Arg(0), "")
}
