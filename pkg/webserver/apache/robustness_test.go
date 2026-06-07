package apache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realWorldConfig 涵盖真实 apache 配置的全部难点：转义引号、引号路径含空格、
// 多层嵌套块、正则、<If> 表达式含尖括号、SSLCipherSuite、续行符、SetHandler 管道
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

// TestRoundTripSemantics 验证综合真实配置 round-trip 语义无损 + 幂等 + 深层嵌套保留
func TestRoundTripSemantics(t *testing.T) {
	cfg, err := ParseString(realWorldConfig)
	require.NoError(t, err)

	out := cfg.Export()
	// 语义无损：原文与导出的指令/块序列（合并续行、分词去引号后）完全一致
	assert.Equal(t, semanticLines(realWorldConfig), semanticLines(out), "语义无损")
	// 幂等
	cfg2, err := ParseString(out)
	require.NoError(t, err)
	assert.Equal(t, out, cfg2.Export(), "幂等")

	// 抽查深层嵌套指令（VirtualHost > Directory > IfModule > RewriteRule）完整保留
	rr := cfg.FindOne("VirtualHost.Directory.IfModule.RewriteRule")
	require.NotNil(t, rr)
	assert.Equal(t, []string{"^(.*)$", "https://new.example.com$1", "[R=301,L]"}, argValues(rr.Args))
}

// TestRobustnessTrickySyntax 针对真实 apache 配置里最刁钻的语法做精确断言
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
		cfg, err := ParseString(c.input)
		require.NoError(t, err, c.name)
		d := cfg.Get(c.dir)
		require.NotNil(t, d, c.name)
		assert.Equal(t, c.wantArgs, argValues(d.Args), c.name)

		// round-trip 后值仍一致
		cfg2, err := ParseString(cfg.Export())
		require.NoError(t, err, c.name+" 重解析")
		assert.Equal(t, c.wantArgs, argValues(cfg2.Get(c.dir).Args), c.name+" round-trip")
	}
}

// TestRobustnessEdgeCases 构造的边界地狱：专攻解析器最易崩的语法
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
		require.NoError(t, err)
		ra := cfg.VirtualHosts()[0].GetBlock("Directory").GetBlock("Files").
			GetBlock("IfModule").GetBlock("Limit").GetBlock("RequireAll")
		require.NotNil(t, ra)
		assert.Equal(t, []string{"all", "granted"}, ra.Values("Require"))
		assert.Equal(t, cfg.Export(), mustReparse(t, cfg.Export()).Export(), "幂等")
	})

	t.Run("If表达式含尖括号", func(t *testing.T) {
		input := "<If \"%{QUERY_STRING} =~ /(>|<)/\">\n    Require all denied\n</If>"
		cfg, err := ParseString(input)
		require.NoError(t, err)
		blocks := cfg.FindBlocks("If")
		require.Len(t, blocks, 1)
		assert.Equal(t, `%{QUERY_STRING} =~ /(>|<)/`, blocks[0].Args[0].Value)
		assert.Equal(t, cfg.Export(), mustReparse(t, cfg.Export()).Export(), "幂等")
	})

	t.Run("CRLF行尾", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\r\n    Require all granted\r\n</Directory>\r\n")
		require.NoError(t, err)
		assert.Equal(t, []string{"all", "granted"}, cfg.GetBlock("Directory").Values("Require"))
	})

	t.Run("未闭合块容错", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n    Require all granted")
		require.NoError(t, err)
		require.NotNil(t, cfg.GetBlock("Directory"))
		assert.Equal(t, []string{"all", "granted"}, cfg.GetBlock("Directory").Values("Require"))
	})

	t.Run("闭合标签大小写不匹配", func(t *testing.T) {
		cfg, err := ParseString("<directory /a>\n    Require all granted\n</Directory>")
		require.NoError(t, err)
		require.NotNil(t, cfg.GetBlock("directory"))
		assert.Equal(t, []string{"all", "granted"}, cfg.GetBlock("directory").Values("Require"))
	})

	t.Run("空块", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n</Directory>")
		require.NoError(t, err)
		require.NotNil(t, cfg.GetBlock("Directory"))
		assert.Empty(t, cfg.GetBlock("Directory").Nodes)
	})

	t.Run("制表符缩进", func(t *testing.T) {
		cfg, err := ParseString("<Directory /a>\n\t\tRequire all granted\n</Directory>")
		require.NoError(t, err)
		assert.Equal(t, []string{"all", "granted"}, cfg.GetBlock("Directory").Values("Require"))
	})

	t.Run("只有注释", func(t *testing.T) {
		cfg, err := ParseString("# c1\n# c2\n")
		require.NoError(t, err)
		assert.Len(t, cfg.Nodes, 2)
	})

	t.Run("纯空白配置", func(t *testing.T) {
		cfg, err := ParseString("\n\n   \n\t\n")
		require.NoError(t, err)
		assert.Empty(t, cfg.Nodes)
	})

	t.Run("块标签引号路径含空格", func(t *testing.T) {
		cfg, err := ParseString("<Directory \"/var/www/my site\">\n    Require all granted\n</Directory>")
		require.NoError(t, err)
		d := cfg.GetBlock("Directory")
		require.NotNil(t, d)
		assert.Equal(t, "/var/www/my site", d.Args[0].Value)
		assert.Contains(t, cfg.Export(), `<Directory "/var/www/my site">`)
	})
}

func mustReparse(t *testing.T, s string) *Config {
	cfg, err := ParseString(s)
	require.NoError(t, err)
	return cfg
}

// semanticLines 把配置规范化为指令/块序列（合并续行、分词去引号、去注释空行），用于语义无损对比
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

func joinValues(args []Argument) string {
	vals := make([]string, len(args))
	for i, a := range args {
		vals[i] = a.Value
	}
	return strings.Join(vals, " ")
}
