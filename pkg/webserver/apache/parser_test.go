package apache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirective(t *testing.T) {
	cfg, err := ParseString("ServerName www.example.com")
	require.NoError(t, err)

	d := cfg.Get("ServerName")
	require.NotNil(t, d)
	assert.Equal(t, "ServerName", d.Name)
	assert.Equal(t, []string{"www.example.com"}, argValues(d.Args))
}

func TestParseMultipleArgs(t *testing.T) {
	cfg, err := ParseString("Listen 192.168.1.100:80")
	require.NoError(t, err)
	assert.Equal(t, []string{"192.168.1.100:80"}, argValues(cfg.Get("Listen").Args))
}

func TestParseQuotedArgs(t *testing.T) {
	cfg, err := ParseString(`CustomLog "/var/log/apache2/access.log" combined`)
	require.NoError(t, err)

	d := cfg.Get("CustomLog")
	require.NotNil(t, d)
	// 引号被解析，值不含引号
	assert.Equal(t, "/var/log/apache2/access.log", d.Args[0].Value)
	assert.Equal(t, QuoteDouble, d.Args[0].Quote)
	assert.Equal(t, "combined", d.Args[1].Value)
	assert.Equal(t, QuoteNone, d.Args[1].Quote)
	// 导出时引号按原风格还原
	assert.Contains(t, cfg.Export(), `CustomLog "/var/log/apache2/access.log" combined`)
}

func TestParseVirtualHost(t *testing.T) {
	input := `<VirtualHost *:80>
    ServerName www.example.com
    DocumentRoot /var/www/html
</VirtualHost>`

	cfg, err := ParseString(input)
	require.NoError(t, err)

	vhosts := cfg.VirtualHosts()
	require.Len(t, vhosts, 1)
	assert.Equal(t, []string{"*:80"}, vhosts[0].ArgValues())
	assert.Equal(t, "www.example.com", vhosts[0].Value("ServerName"))
	assert.Equal(t, "/var/www/html", vhosts[0].Value("DocumentRoot"))
}

// TestNestedBlocksTriple 验证三层嵌套块正确解析
func TestNestedBlocksTriple(t *testing.T) {
	input := `<VirtualHost *:80>
    <Directory /var/www>
        <Files index.php>
            Require all granted
        </Files>
    </Directory>
</VirtualHost>`

	cfg, err := ParseString(input)
	require.NoError(t, err)

	dir := cfg.VirtualHosts()[0].GetBlock("Directory")
	require.NotNil(t, dir)
	files := dir.GetBlock("Files")
	require.NotNil(t, files)
	assert.Equal(t, []string{"all", "granted"}, files.Values("Require"))
}

// TestNestedIfModuleNotDropped 验证双层嵌套块不被丢弃（旧实现的头号 bug）
func TestNestedIfModuleNotDropped(t *testing.T) {
	input := `<IfModule mod_proxy_balancer.c>
    <Proxy balancer://backend>
        BalancerMember http://127.0.0.1:8080 loadfactor=5
        BalancerMember http://127.0.0.1:8081
        ProxySet lbmethod=byrequests
    </Proxy>
</IfModule>`

	cfg, err := ParseString(input)
	require.NoError(t, err)

	members := cfg.Find("IfModule.Proxy.BalancerMember")
	require.Len(t, members, 2)
	assert.Equal(t, "http://127.0.0.1:8080", members[0].Args[0].Value)
	assert.Equal(t, "loadfactor=5", members[0].Args[1].Value)
}

// TestTokenizeSpecialChars 验证特殊字符不被分词器粘连或截断（旧 lexer 的字符白名单 bug）
func TestTokenizeSpecialChars(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{`RewriteCond %{HTTP_HOST} ^old\.example\.com$ [NC]`, []string{"%{HTTP_HOST}", `^old\.example\.com$`, "[NC]"}},
		{`RewriteRule ^(.*)$ https://new.example.com$1 [R=308,L]`, []string{"^(.*)$", "https://new.example.com$1", "[R=308,L]"}},
		{`ProxyPass / http://127.0.0.1:8080/app`, []string{"/", "http://127.0.0.1:8080/app"}},
		{`SetHandler "proxy:unix:/tmp/php-cgi-84.sock|fcgi://localhost/"`, []string{"proxy:unix:/tmp/php-cgi-84.sock|fcgi://localhost/"}},
	}
	for _, c := range cases {
		cfg, err := ParseString(c.input)
		require.NoError(t, err, c.input)
		d, ok := cfg.Nodes[0].(*Directive)
		require.True(t, ok, c.input)
		assert.Equal(t, c.want, argValues(d.Args), c.input)
	}
}

// TestLineContinuation 验证续行符合并
func TestLineContinuation(t *testing.T) {
	cfg, err := ParseString("RewriteCond %{HTTP_HOST} foo \\\n    [NC]")
	require.NoError(t, err)
	d := cfg.Get("RewriteCond")
	require.NotNil(t, d)
	assert.Equal(t, []string{"%{HTTP_HOST}", "foo", "[NC]"}, argValues(d.Args))
}

// TestLineContinuationEscapedBackslash 验证偶数反斜杠不触发续行
func TestLineContinuationEscapedBackslash(t *testing.T) {
	cfg, err := ParseString("ServerName a\\\\\nServerAdmin b")
	require.NoError(t, err)
	assert.NotNil(t, cfg.Get("ServerName"))
	assert.NotNil(t, cfg.Get("ServerAdmin"))
}

// TestCommentSemantics 验证注释语义：整行注释 vs 行内 #
func TestCommentSemantics(t *testing.T) {
	cfg, err := ParseString("# a comment\nServerName x")
	require.NoError(t, err)
	cmts := collectComments(cfg.Nodes)
	require.Len(t, cmts, 1)
	assert.Equal(t, " a comment", cmts[0].Text)

	// 行内 # 是参数的一部分，不当注释
	cfg2, err := ParseString("Redirect 301 /a /b#frag")
	require.NoError(t, err)
	d := cfg2.Get("Redirect")
	require.NotNil(t, d)
	assert.Equal(t, "/b#frag", d.Args[2].Value)
}

// TestCommentPreserveLeadingSpace 验证注释首空格不丢失
func TestCommentPreserveLeadingSpace(t *testing.T) {
	cfg, err := ParseString("#  double space")
	require.NoError(t, err)
	assert.Contains(t, cfg.Export(), "#  double space")
}

// TestRoundTripDefaultVhostConf 验证默认模板规范化幂等且 IncludeOptional 在 VirtualHost 前
func TestRoundTripDefaultVhostConf(t *testing.T) {
	cfg, err := ParseString(DefaultVhostConf)
	require.NoError(t, err)

	rendered := cfg.Render()
	cfg2, err := ParseString(rendered)
	require.NoError(t, err)
	assert.Equal(t, rendered, cfg2.Render(), "规范化导出应幂等")

	idxInclude := strings.Index(rendered, "IncludeOptional")
	idxVhost := strings.Index(rendered, "<VirtualHost")
	require.GreaterOrEqual(t, idxInclude, 0)
	require.GreaterOrEqual(t, idxVhost, 0)
	assert.Less(t, idxInclude, idxVhost, "顶层 IncludeOptional 应排在 VirtualHost 之前")
}

// TestExportNestedRoundTrip 验证嵌套块导出后可再次解析为等价结构
func TestExportNestedRoundTrip(t *testing.T) {
	input := `<VirtualHost *:80>
    ServerName x
    <Directory /var/www>
        Require all granted
    </Directory>
</VirtualHost>`

	cfg, err := ParseString(input)
	require.NoError(t, err)

	out := cfg.Export()
	cfg2, err := ParseString(out)
	require.NoError(t, err)
	assert.Equal(t, out, cfg2.Export(), "保序导出应幂等")
	assert.NotNil(t, cfg2.VirtualHosts()[0].GetBlock("Directory"))
}

func TestQueryCaseInsensitive(t *testing.T) {
	cfg, err := ParseString("ServerName x")
	require.NoError(t, err)
	assert.NotNil(t, cfg.Get("servername"))
	assert.Equal(t, "x", cfg.Value("SERVERNAME"))
	assert.True(t, cfg.Has("ServerName"))
}

func TestFindDotPath(t *testing.T) {
	input := `<IfModule a.c>
    <Proxy p>
        Member 1
        Member 2
    </Proxy>
</IfModule>`

	cfg, err := ParseString(input)
	require.NoError(t, err)
	assert.Len(t, cfg.Find("IfModule.Proxy.Member"), 2)
	assert.Len(t, cfg.FindBlocks("IfModule.Proxy"), 1)
	assert.Nil(t, cfg.FindOne("IfModule.Proxy.Missing"))
}

// TestTolerantUnclosedBlock 验证未闭合块在容错模式下不致命
func TestTolerantUnclosedBlock(t *testing.T) {
	cfg, err := ParseString("<VirtualHost *:80>\n    ServerName x")
	require.NoError(t, err)
	require.Len(t, cfg.VirtualHosts(), 1)
	assert.Equal(t, "x", cfg.VirtualHosts()[0].Value("ServerName"))
}

// TestTolerantOrphanCloseTag 验证孤立闭合标签在容错模式下被跳过
func TestTolerantOrphanCloseTag(t *testing.T) {
	cfg, err := ParseString("</Foo>\nServerName x")
	require.NoError(t, err)
	assert.Equal(t, "x", cfg.Value("ServerName"))
}

func TestParseEmpty(t *testing.T) {
	cfg, err := ParseString("")
	require.NoError(t, err)
	assert.Empty(t, cfg.Nodes)
}

func TestSetAndRemove(t *testing.T) {
	cfg, err := ParseString("ServerName old")
	require.NoError(t, err)

	cfg.Set("ServerName", "new")
	assert.Equal(t, "new", cfg.Value("ServerName"))

	cfg.Add("ServerAlias", "a", "b")
	assert.Equal(t, []string{"a", "b"}, cfg.Values("ServerAlias"))

	assert.True(t, cfg.Remove("ServerName"))
	assert.False(t, cfg.Has("ServerName"))
}

// TestAddDirectiveAutoQuote 验证 Add 对含空格的参数自动加引号
func TestAddDirectiveAutoQuote(t *testing.T) {
	cfg := &Config{}
	cfg.Add("AuthName", "My Realm")
	assert.Contains(t, cfg.Export(), `AuthName "My Realm"`)

	cfg2 := &Config{}
	cfg2.Add("DocumentRoot", "/var/www")
	assert.Contains(t, cfg2.Export(), "DocumentRoot /var/www")
	assert.NotContains(t, cfg2.Export(), `"`)
}
