package nginx

import (
	"fmt"
	"os"
	"strings"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

const phpCacheConf = `# browser cache
location ~ .*\.(bmp|jpg|jpeg|png|gif|svg|ico|tiff|webp|avif|heif|heic|jxl)$ {
    expires 30d;
    access_log /dev/null;
    error_log /dev/null;
}
location ~ .*\.(js|css|ttf|otf|woff|woff2|eot)$ {
    expires 6h;
    access_log /dev/null;
    error_log /dev/null;
}
# deny sensitive files
location ~ ^/(\.user.ini|\.htaccess|\.git|\.svn|\.env) {
    return 404;
}
`

const spaConf = `# single-page application route fallback, remove if not needed
location / {
    try_files $uri $uri/ /index.html;
}
`

// Dialect Nginx 方言
type Dialect struct{}

func (Dialect) Service() string {
	return "nginx"
}

func (Dialect) ConfigTest() string {
	return "nginx -t 2>&1"
}

func (Dialect) HTMLDir() string {
	return HTMLDir
}

func (Dialect) ConfigFile() string {
	return "nginx.conf"
}

func (Dialect) PanelACMEConf() string {
	return "/opt/ace/server/nginx/conf/acme.conf"
}

func (Dialect) Features() types.Features {
	return types.Features{IPv6Listen: true, Stat: true, DefaultSite: true}
}

func (Dialect) HTTPSListenArgs() []string {
	return []string{"ssl", "quic"}
}

func (Dialect) ErrorPageConf() string {
	return "error_page 404 /404.html;"
}

func (Dialect) PHPCacheConf() string {
	return phpCacheConf
}

func (Dialect) SPAConf() string {
	return spaConf
}

func (Dialect) HTPasswdLine(username, password string) string {
	return username + ":{PLAIN}" + password
}

func (Dialect) RewritesDir() string {
	return "nginx"
}

func (Dialect) BeforeReload() error {
	return nil
}

func (Dialect) NewStaticVhost(configDir string) (types.StaticVhost, error) {
	vhost, err := NewStaticVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

func (Dialect) NewPHPVhost(configDir string) (types.PHPVhost, error) {
	vhost, err := NewPHPVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

func (Dialect) NewProxyVhost(configDir string) (types.ProxyVhost, error) {
	vhost, err := NewProxyVhost(configDir)
	if err != nil {
		return nil, err
	}
	return vhost, nil
}

func (Dialect) WriteSiteChallenge(conf, path, token string) error {
	file, err := os.OpenFile(conf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open nginx config %q: %w", conf, err)
	}
	_, err = file.WriteString(challengeConf(path, token))
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("failed to write to nginx config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) RemoveSiteChallenge(conf, path, token string) error {
	raw, err := os.ReadFile(conf)
	if err != nil {
		return fmt.Errorf("failed to read nginx config %q: %w", conf, err)
	}
	content := strings.ReplaceAll(string(raw), challengeConf(path, token), "")
	if err = os.WriteFile(conf, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write to nginx config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) WritePanelChallenge(conf string, names []string, tokens map[string]string) error {
	var b strings.Builder
	b.WriteString("server {\n    listen 80;\n")
	// 只有在包含 IPv6 地址时才监听 [::]:80，避免纯 IPv4 系统上 nginx 启动失败
	if lo.SomeBy(names, tools.IsIPv6) {
		b.WriteString("    listen [::]:80;\n")
	}
	wrapped := lo.Map(names, func(name string, _ int) string {
		return tools.WrapIPv6(name)
	})
	_, _ = fmt.Fprintf(&b, "    server_name %s;\n", strings.Join(wrapped, " "))
	for path, token := range tokens {
		_, _ = fmt.Fprintf(&b, "    location = %s {\n        default_type text/plain;\n        return 200 %q;\n    }\n", path, token)
	}
	b.WriteString("}\n")

	if err := os.WriteFile(conf, []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("failed to write nginx config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) RemovePanelChallenge(conf string) error {
	if err := os.WriteFile(conf, []byte(""), 0600); err != nil {
		return fmt.Errorf("failed to write to config %q: %w", conf, err)
	}

	return nil
}

// challengeConf 单个 HTTP-01 验证的 location 片段
func challengeConf(path, token string) string {
	return fmt.Sprintf("location = %s {\n    default_type text/plain;\n    return 200 %q;\n}\n", path, token)
}
