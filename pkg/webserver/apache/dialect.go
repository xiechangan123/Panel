package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// panelTokenDir 面板证书验证的 token 目录
const panelTokenDir = "/tmp/acme-challenge"

const phpCacheConf = `# browser cache
<IfModule mod_expires.c>
    ExpiresActive On
    ExpiresByType image/bmp "access plus 30 days"
    ExpiresByType image/jpeg "access plus 30 days"
    ExpiresByType image/png "access plus 30 days"
    ExpiresByType image/gif "access plus 30 days"
    ExpiresByType image/svg+xml "access plus 30 days"
    ExpiresByType image/x-icon "access plus 30 days"
    ExpiresByType image/tiff "access plus 30 days"
    ExpiresByType image/webp "access plus 30 days"
    ExpiresByType image/avif "access plus 30 days"
    ExpiresByType image/heif "access plus 30 days"
    ExpiresByType image/heic "access plus 30 days"
    ExpiresByType image/jxl "access plus 30 days"
    ExpiresByType text/css "access plus 6 hours"
    ExpiresByType application/javascript "access plus 6 hours"
    ExpiresByType font/ttf "access plus 6 hours"
    ExpiresByType font/otf "access plus 6 hours"
    ExpiresByType font/woff "access plus 6 hours"
    ExpiresByType font/woff2 "access plus 6 hours"
    ExpiresByType application/vnd.ms-fontobject "access plus 6 hours"
</IfModule>
# deny sensitive files
<FilesMatch "^(\.user\.ini|\.htaccess|\.git|\.svn|\.env)">
    Require all denied
</FilesMatch>
`

const spaConf = `# single-page application route fallback, remove if not needed
FallbackResource /index.html
`

// Dialect Apache 方言
type Dialect struct{}

func (Dialect) Service() string {
	return "apache"
}

func (Dialect) ConfigTest() string {
	return "apachectl configtest 2>&1"
}

func (Dialect) HTMLDir() string {
	return HTMLDir
}

func (Dialect) PanelACMEConf() string {
	return "/opt/ace/server/apache/conf/extra/acme.conf"
}

func (Dialect) Features() types.Features {
	return types.Features{}
}

func (Dialect) HTTPSListenArgs() []string {
	return []string{"ssl"}
}

func (Dialect) ErrorPageConf() string {
	return "ErrorDocument 404 /404.html"
}

func (Dialect) PHPCacheConf() string {
	return phpCacheConf
}

func (Dialect) SPAConf() string {
	return spaConf
}

func (Dialect) HTPasswdLine(username, password string) string {
	return username + ":" + password
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

// WriteSiteChallenge token 落盘到站点配置目录旁的 acme-challenge，再用 Alias 映射
func (Dialect) WriteSiteChallenge(conf, path, token string) error {
	tokenDir := filepath.Join(filepath.Dir(conf), "acme-challenge")
	if err := writeToken(tokenDir, path, token); err != nil {
		return err
	}

	file, err := os.OpenFile(conf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open apache config %q: %w", conf, err)
	}
	_, err = file.WriteString(challengeConf(tokenDir))
	_ = file.Close()
	if err != nil {
		return fmt.Errorf("failed to write to apache config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) RemoveSiteChallenge(conf, path, _ string) error {
	tokenDir := filepath.Join(filepath.Dir(conf), "acme-challenge")
	_ = os.Remove(filepath.Join(tokenDir, filepath.Base(path)))

	raw, err := os.ReadFile(conf)
	if err != nil {
		return fmt.Errorf("failed to read apache config %q: %w", conf, err)
	}
	content := strings.ReplaceAll(string(raw), challengeConf(tokenDir), "")
	if err = os.WriteFile(conf, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write to apache config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) WritePanelChallenge(conf string, names []string, tokens map[string]string) error {
	for path, token := range tokens {
		if err := writeToken(panelTokenDir, path, token); err != nil {
			return err
		}
	}

	wrapped := lo.Map(names, func(name string, _ int) string {
		return tools.WrapIPv6(name)
	})
	var b strings.Builder
	b.WriteString("<VirtualHost *:80>\n")
	_, _ = fmt.Fprintf(&b, "    ServerName %s\n", wrapped[0])
	if len(wrapped) > 1 {
		_, _ = fmt.Fprintf(&b, "    ServerAlias %s\n", strings.Join(wrapped[1:], " "))
	}
	_, _ = fmt.Fprintf(&b, "    Alias /.well-known/acme-challenge %s\n", panelTokenDir)
	_, _ = fmt.Fprintf(&b, "    <Directory %s>\n", panelTokenDir)
	b.WriteString("        Require all granted\n")
	b.WriteString("        ForceType text/plain\n")
	b.WriteString("    </Directory>\n")
	b.WriteString("</VirtualHost>\n")

	if err := os.WriteFile(conf, []byte(b.String()), 0600); err != nil {
		return fmt.Errorf("failed to write apache config %q: %w", conf, err)
	}

	return nil
}

func (Dialect) RemovePanelChallenge(conf string) error {
	if err := os.WriteFile(conf, []byte(""), 0600); err != nil {
		return fmt.Errorf("failed to write to config %q: %w", conf, err)
	}
	_ = os.RemoveAll(panelTokenDir)

	return nil
}

// writeToken 将验证 token 写入目录，文件名取 URL 路径的最后一段
func writeToken(dir, path, token string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(path)), []byte(token), 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// challengeConf 把 /.well-known/acme-challenge 映射到 token 目录的片段
func challengeConf(tokenDir string) string {
	return fmt.Sprintf(`Alias /.well-known/acme-challenge %s
<Directory %s>
    Require all granted
    ForceType text/plain
</Directory>
`, tokenDir, tokenDir)
}
