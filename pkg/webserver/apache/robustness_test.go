package apache

import (
	"strings"
	"testing"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

// realWorldConfig 汇集真实 apache 配置里最难解析的语法
const realWorldConfig = `# main config
ServerRoot "/etc/httpd"
Listen 80
LoadModule ssl_module modules/mod_ssl.so

<VirtualHost *:443>
    ServerName www.example.com
    ServerAlias example.com *.example.com
    DocumentRoot "/var/www/my site"
    DirectoryIndex index.html index.php
    LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\"" combined
    CustomLog "/var/log/access.log" combined
    <Directory "/var/www/my site">
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
        <IfModule mod_rewrite.c>
            RewriteEngine On
            RewriteCond %{HTTP_HOST} ^old\.example\.com$ [NC]
            RewriteRule ^(.*)$ https://new.example.com$1 [R=301,L]
        </IfModule>
    </Directory>
    <If "%{QUERY_STRING} =~ /(<|>)/">
        Require all denied
    </If>
    SSLEngine on
    SSLCipherSuite HIGH:MEDIUM:!MD5:!RC4:!3DES
    SSLOpenSSLConfCmd Options \
        -SessionTicket
    <FilesMatch "\.php$">
        SetHandler "proxy:unix:/tmp/php-cgi-84.sock|fcgi://localhost/"
    </FilesMatch>
</VirtualHost>`

func TestRoundTripSemantics(t *testing.T) {
	cfg, err := ParseString(realWorldConfig)
	must.NoError(t, err)

	out := Export(cfg)
	check.DeepEqual(t, semanticLines(out), semanticLines(realWorldConfig))
	cfg2, err := ParseString(out)
	must.NoError(t, err)
	check.Equal(t, Export(cfg2), out, check.Msgf("幂等"))

	rr := cfg.FindOne("VirtualHost.Directory.IfModule.RewriteRule")
	must.NotNil(t, rr)
	check.DeepEqual(t, rr.Values(), []string{"^(.*)$", "https://new.example.com$1", "[R=301,L]"})
}

func TestRobustnessTrickySyntax(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		dir      string
		wantArgs []string
	}{
		{
			"LogFormat 嵌套转义引号",
			`LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\"" combined`,
			"LogFormat",
			[]string{`%h %l %u %t "%r" %>s %b "%{Referer}i"`, "combined"},
		},
		{
			"SSLCipherSuite 含冒号叹号",
			`SSLCipherSuite HIGH:MEDIUM:!MD5:!RC4:!3DES`,
			"SSLCipherSuite",
			[]string{"HIGH:MEDIUM:!MD5:!RC4:!3DES"},
		},
		{
			"AliasMatch 复杂正则+引号参数",
			`AliasMatch ^/manual(?:/(?:da|de|en))?(/.*)?$ "@exp_manualdir@$1"`,
			"AliasMatch",
			[]string{`^/manual(?:/(?:da|de|en))?(/.*)?$`, "@exp_manualdir@$1"},
		},
		{
			"tab 分隔多参数",
			"ProxyHTMLLinks\ta\t\thref",
			"ProxyHTMLLinks",
			[]string{"a", "href"},
		},
		{
			"续行符合并",
			"SSLOpenSSLConfCmd Options \\\n    -SessionTicket",
			"SSLOpenSSLConfCmd",
			[]string{"Options", "-SessionTicket"},
		},
		{
			"单引号参数",
			`SetEnvIf User-Agent '^Mozilla' is_mozilla`,
			"SetEnvIf",
			[]string{"User-Agent", "^Mozilla", "is_mozilla"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := ParseString(c.input)
			must.NoError(t, err)
			d := cfg.Get(c.dir)
			must.NotNil(t, d)
			check.DeepEqual(t, d.Values(), c.wantArgs)

			cfg2, err := ParseString(Export(cfg))
			must.NoError(t, err)
			check.DeepEqual(t, cfg2.Get(c.dir).Values(), c.wantArgs)
		})
	}
}

func TestRobustnessEdgeCases(t *testing.T) {
	t.Run("六层深嵌套", func(t *testing.T) {
		input := `<VirtualHost *:80>
  <Directory /a>
    <Files x>
      <IfModule m>
        <Limit GET>
          <RequireAll>
            Require all granted
          </RequireAll>
        </Limit>
      </IfModule>
    </Files>
  </Directory>
</VirtualHost>`
		cfg, err := ParseString(input)
		must.NoError(t, err)
		vhost := cfg.GetBlock("VirtualHost")
		must.NotNil(t, vhost)
		ra := vhost.GetBlock("Directory").GetBlock("Files").
			GetBlock("IfModule").GetBlock("Limit").GetBlock("RequireAll")
		must.NotNil(t, ra)
		check.DeepEqual(t, ra.Get("Require").Values(), []string{"all", "granted"})
		out := Export(cfg)
		check.Equal(t, Export(mustReparse(t, out)), out, check.Msgf("幂等"))
	})

	t.Run("If表达式含尖括号", func(t *testing.T) {
		input := "<If \"%{QUERY_STRING} =~ /(>|<)/\">\n    Require all denied\n</If>"
		cfg, err := ParseString(input)
		must.NoError(t, err)
		blocks := cfg.FindBlocks("If")
		must.Len(t, blocks, 1)
		check.DeepEqual(t, blocks[0].Values(), []string{`%{QUERY_STRING} =~ /(>|<)/`})
		out := Export(cfg)
		check.Equal(t, Export(mustReparse(t, out)), out, check.Msgf("幂等"))
	})

	t.Run("CRLF行尾", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\r\n    Require all granted\r\n</Directory>\r\n")
		must.NoError(t, err)
		check.DeepEqual(t, cfg.GetBlock("Directory").Get("Require").Values(), []string{"all", "granted"})
	})

	t.Run("未闭合块容错", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n    Require all granted")
		must.NoError(t, err)
		must.NotNil(t, cfg.GetBlock("Directory"))
		check.DeepEqual(t, cfg.GetBlock("Directory").Get("Require").Values(), []string{"all", "granted"})
	})

	t.Run("闭合标签大小写不匹配", func(t *testing.T) {
		cfg, err := ParseString("<directory /a>\n    Require all granted\n</Directory>")
		must.NoError(t, err)
		must.NotNil(t, cfg.GetBlock("directory"))
		check.DeepEqual(t, cfg.GetBlock("directory").Get("Require").Values(), []string{"all", "granted"})
	})

	t.Run("空块", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n</Directory>")
		must.NoError(t, err)
		must.NotNil(t, cfg.GetBlock("Directory"))
		check.Empty(t, cfg.GetBlock("Directory").Nodes)
	})

	t.Run("制表符缩进", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n\t\tRequire all granted\n</Directory>")
		must.NoError(t, err)
		check.DeepEqual(t, cfg.GetBlock("Directory").Get("Require").Values(), []string{"all", "granted"})
	})

	t.Run("只有注释", func(t *testing.T) {
		cfg, err := ParseString("# c1\n# c2\n")
		must.NoError(t, err)
		check.Len(t, cfg.Nodes, 2)
		cmts := collectComments(cfg.Nodes)
		must.Len(t, cmts, 2)
		check.DeepEqual(t, []string{cmts[0].Text, cmts[1].Text}, []string{" c1", " c2"})
	})

	t.Run("纯空白配置", func(t *testing.T) {
		cfg, err := ParseString("\n\n   \n\t\n")
		must.NoError(t, err)
		check.Empty(t, cfg.Nodes)
	})

	t.Run("块标签引号路径含空格", func(t *testing.T) {
		cfg, err := ParseString("<Directory \"/var/www/my site\">\n    Require all granted\n</Directory>")
		must.NoError(t, err)
		d := cfg.GetBlock("Directory")
		must.NotNil(t, d)
		check.DeepEqual(t, d.Values(), []string{"/var/www/my site"})
		check.Contains(t, Export(cfg), `<Directory "/var/www/my site">`)
	})
}

func mustReparse(t *testing.T, s string) *conf.Config {
	t.Helper()
	cfg, err := ParseString(s)
	must.NoError(t, err)
	return cfg
}

// semanticLines 把配置归一成指令/块序列，用于比对语义是否无损
func semanticLines(src string) []string {
	var out []string
	for _, ln := range scanLogicalLines(src) {
		switch {
		case strings.HasPrefix(ln, "#"):
			continue
		case strings.HasPrefix(ln, "</"):
			out = append(out, "</"+parseCloseTag(ln)+">")
		case strings.HasPrefix(ln, "<"):
			name, argStr := parseOpenTag(ln)
			out = append(out, "<"+name+" "+joinValues(tokenizeLine(argStr))+">")
		default:
			out = append(out, joinValues(tokenizeLine(ln)))
		}
	}
	return out
}

func joinValues(args []conf.Arg) string {
	vals := make([]string, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	return strings.Join(vals, " ")
}
