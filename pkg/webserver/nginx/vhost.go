package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/acepanel/panel/pkg/webserver/types"
)

// StaticVhost 纯静态虚拟主机
type StaticVhost struct {
	*baseVhost
}

// PHPVhost PHP 虚拟主机
type PHPVhost struct {
	*baseVhost
}

// ProxyVhost 反向代理虚拟主机
type ProxyVhost struct {
	*baseVhost
}

// baseVhost Nginx 虚拟主机基础实现
type baseVhost struct {
	parser    *Parser
	configDir string // 配置目录
}

// newBaseVhost 创建基础虚拟主机实例
func newBaseVhost(configDir string) (*baseVhost, error) {
	if configDir == "" {
		return nil, fmt.Errorf("config directory is required")
	}

	v := &baseVhost{
		configDir: configDir,
	}

	// 加载配置
	var parser *Parser
	var err error

	// 从配置目录加载主配置文件
	configFile := filepath.Join(v.configDir, "nginx.conf")
	if _, statErr := os.Stat(configFile); statErr == nil {
		parser, err = NewParserFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load nginx config: %w", err)
		}
	}

	// 如果没有配置文件，使用默认配置
	if parser == nil {
		// 使用空字符串创建默认配置，而不尝试读取文件
		parser, err = NewParser("")
		if err != nil {
			return nil, fmt.Errorf("failed to load default config: %w", err)
		}
		parser.SetConfigPath(filepath.Join(v.configDir, "nginx.conf"))
	}

	v.parser = parser
	return v, nil
}

// NewStaticVhost 创建纯静态虚拟主机实例
func NewStaticVhost(configDir string) (*StaticVhost, error) {
	base, err := newBaseVhost(configDir)
	if err != nil {
		return nil, err
	}
	return &StaticVhost{baseVhost: base}, nil
}

// NewPHPVhost 创建 PHP 虚拟主机实例
func NewPHPVhost(configDir string) (*PHPVhost, error) {
	base, err := newBaseVhost(configDir)
	if err != nil {
		return nil, err
	}
	return &PHPVhost{baseVhost: base}, nil
}

// NewProxyVhost 创建反向代理虚拟主机实例
func NewProxyVhost(configDir string) (*ProxyVhost, error) {
	base, err := newBaseVhost(configDir)
	if err != nil {
		return nil, err
	}
	return &ProxyVhost{baseVhost: base}, nil
}

func (v *baseVhost) Enable() bool {
	// 检查禁用配置文件是否存在
	disableFile := filepath.Join(v.configDir, "vhost", DisableConfName)
	_, err := os.Stat(disableFile)
	return os.IsNotExist(err)
}

func (v *baseVhost) SetEnable(enable bool, _ ...string) error {
	serverDir := filepath.Join(v.configDir, "vhost")
	disableFile := filepath.Join(serverDir, DisableConfName)

	if enable {
		// 启用：删除禁用配置文件
		if err := os.Remove(disableFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove disable config: %w", err)
		}
		return nil
	}

	// 禁用：创建禁用配置文件
	if err := os.WriteFile(disableFile, []byte(DisableConfContent), 0644); err != nil {
		return fmt.Errorf("failed to write disable config: %w", err)
	}

	return nil
}

func (v *baseVhost) Listen() []types.Listen {
	listens, err := v.parser.GetListen()
	if err != nil {
		return nil
	}

	var result []types.Listen
	for _, l := range listens {
		if len(l) == 0 {
			continue
		}

		listen := types.Listen{
			Address: l[0],
			Options: make(map[string]string),
		}

		// 解析 Nginx 特有的选项
		for i := 1; i < len(l); i++ {
			switch l[i] {
			case "ssl":
				listen.Protocol = "https"
			case "http2":
				listen.Protocol = "http2"
			case "http3", "quic":
				listen.Protocol = "http3"
			default:
				listen.Options[l[i]] = "true"
			}
		}

		// 如果没有指定协议，默认为 http
		if listen.Protocol == "" {
			listen.Protocol = "http"
		}

		result = append(result, listen)
	}

	return result
}

func (v *baseVhost) SetListen(listens []types.Listen) error {
	// 将通用 Listen 转换为 Nginx 格式
	var nginxListens [][]string
	for _, l := range listens {
		listen := []string{l.Address}

		// 添加协议标识
		switch l.Protocol {
		case "https":
			listen = append(listen, "ssl")
		case "http2":
			listen = append(listen, "http2")
		case "http3":
			listen = append(listen, "http3")
		}

		// 添加其他选项
		for k, v := range l.Options {
			if v == "true" {
				listen = append(listen, k)
			} else {
				listen = append(listen, fmt.Sprintf("%s=%s", k, v))
			}
		}

		nginxListens = append(nginxListens, listen)
	}

	return v.parser.SetListen(nginxListens)
}

func (v *baseVhost) ServerName() []string {
	names, err := v.parser.GetServerName()
	if err != nil {
		return nil
	}
	return names
}

func (v *baseVhost) SetServerName(serverName []string) error {
	return v.parser.SetServerName(serverName)
}

func (v *baseVhost) Index() []string {
	index, err := v.parser.GetIndex()
	if err != nil {
		return nil
	}
	return index
}

func (v *baseVhost) SetIndex(index []string) error {
	return v.parser.SetIndex(index)
}

func (v *baseVhost) Root() string {
	root, err := v.parser.GetRoot()
	if err != nil {
		return ""
	}
	return root
}

func (v *baseVhost) SetRoot(root string) error {
	return v.parser.SetRoot(root)
}

func (v *baseVhost) Includes() []types.IncludeFile {
	includes, comments, err := v.parser.GetIncludes()
	if err != nil {
		return nil
	}

	var result []types.IncludeFile
	for i, inc := range includes {
		file := types.IncludeFile{
			Path: inc,
		}
		if i < len(comments) {
			file.Comment = comments[i]
		}
		result = append(result, file)
	}

	return result
}

func (v *baseVhost) SetIncludes(includes []types.IncludeFile) error {
	var paths []string
	var comments [][]string

	for _, inc := range includes {
		paths = append(paths, inc.Path)
		comments = append(comments, inc.Comment)
	}

	return v.parser.SetIncludes(paths, comments)
}

func (v *baseVhost) AccessLog() string {
	log, err := v.parser.GetAccessLog()
	if err != nil {
		return ""
	}
	return log
}

func (v *baseVhost) SetAccessLog(accessLog string) error {
	return v.parser.SetAccessLog(accessLog)
}

func (v *baseVhost) ErrorLog() string {
	log, err := v.parser.GetErrorLog()
	if err != nil {
		return ""
	}
	return log
}

func (v *baseVhost) SetErrorLog(errorLog string) error {
	return v.parser.SetErrorLog(errorLog)
}

func (v *baseVhost) Save() error {
	return v.parser.Save()
}

func (v *baseVhost) Reload() error {
	parts := strings.Fields("systemctl reload openresty")
	if err := exec.Command(parts[0], parts[1:]...).Run(); err != nil {
		return fmt.Errorf("failed to reload nginx config: %w", err)
	}

	return nil
}

func (v *baseVhost) Reset() error {
	// 重置配置为默认值
	parser, err := NewParser("")
	if err != nil {
		return fmt.Errorf("failed to reset config: %w", err)
	}

	// 如果有 configDir，设置配置文件路径
	if v.configDir != "" {
		parser.SetConfigPath(filepath.Join(v.configDir, "nginx.conf"))
	}

	v.parser = parser
	return nil
}

func (v *baseVhost) HTTPS() bool {
	return v.parser.GetHTTPS()
}

func (v *baseVhost) SSLConfig() *types.SSLConfig {
	if !v.HTTPS() {
		return nil
	}

	return &types.SSLConfig{
		Protocols:    v.parser.GetHTTPSProtocols(),
		Ciphers:      v.parser.GetHTTPSCiphers(),
		HSTS:         v.parser.GetHSTS(),
		OCSP:         v.parser.GetOCSP(),
		HTTPRedirect: v.parser.GetHTTPSRedirect(),
		AltSvc:       v.parser.GetAltSvc(),
	}
}

func (v *baseVhost) SetSSLConfig(cfg *types.SSLConfig) error {
	if cfg == nil {
		return fmt.Errorf("SSL config cannot be nil")
	}

	// 设置证书和私钥
	if err := v.parser.SetHTTPSCert(cfg.Cert, cfg.Key); err != nil {
		return err
	}

	// 设置协议
	if len(cfg.Protocols) > 0 {
		if err := v.parser.SetHTTPSProtocols(cfg.Protocols); err != nil {
			return err
		}
	}

	// 设置加密套件
	if cfg.Ciphers != "" {
		if err := v.parser.SetHTTPSCiphers(cfg.Ciphers); err != nil {
			return err
		}
	}

	// 设置 HSTS
	if err := v.parser.SetHSTS(cfg.HSTS); err != nil {
		return err
	}

	// 设置 OCSP
	if err := v.parser.SetOCSP(cfg.OCSP); err != nil {
		return err
	}

	// 设置 HTTP 跳转
	if err := v.parser.SetHTTPSRedirect(cfg.HTTPRedirect); err != nil {
		return err
	}

	// 设置 Alt-Svc
	if cfg.AltSvc != "" {
		if err := v.parser.SetAltSvc(cfg.AltSvc); err != nil {
			return err
		}
	}

	return nil
}

func (v *baseVhost) ClearHTTPS() error {
	return v.parser.ClearHTTPS()
}

func (v *baseVhost) RateLimit() *types.RateLimit {
	rate := v.parser.GetLimitRate()
	limitConn := v.parser.GetLimitConn()

	if rate == "" && len(limitConn) == 0 {
		return nil
	}

	rateLimit := &types.RateLimit{
		Rate:    rate,
		Options: make(map[string]string),
	}

	// 解析 limit_conn 配置
	for _, limit := range limitConn {
		if len(limit) >= 2 {
			// limit_conn zone connections
			// 例如: limit_conn perip 10
			rateLimit.Options[limit[0]] = limit[1]
		}
	}

	return rateLimit
}

func (v *baseVhost) SetRateLimit(limit *types.RateLimit) error {
	if limit == nil {
		// 清除限流配置
		if err := v.parser.SetLimitRate(""); err != nil {
			return err
		}
		return v.parser.SetLimitConn(nil)
	}

	// 设置限速
	if err := v.parser.SetLimitRate(limit.Rate); err != nil {
		return err
	}

	// 设置并发连接数限制
	var limitConns [][]string
	for zone, connections := range limit.Options {
		limitConns = append(limitConns, []string{zone, connections})
	}

	return v.parser.SetLimitConn(limitConns)
}

func (v *baseVhost) BasicAuth() map[string]string {
	realm, userFile := v.parser.GetBasicAuth()
	if realm == "" || userFile == "" {
		return nil
	}

	// 返回基本认证配置
	// 注意：这里只返回配置路径，不解析用户文件内容
	return map[string]string{
		"realm":     realm,
		"user_file": userFile,
	}
}

func (v *baseVhost) SetBasicAuth(auth map[string]string) error {
	if len(auth) == 0 {
		// 清除基本认证配置
		return v.parser.SetBasicAuth("", "")
	}

	realm := auth["realm"]
	userFile := auth["user_file"]

	if realm == "" {
		realm = "Restricted"
	}

	return v.parser.SetBasicAuth(realm, userFile)
}

func (v *baseVhost) Redirects() []types.Redirect {
	vhostDir := filepath.Join(v.configDir, "vhost")
	redirects, _ := parseRedirectFiles(vhostDir)
	return redirects
}

func (v *baseVhost) SetRedirects(redirects []types.Redirect) error {
	vhostDir := filepath.Join(v.configDir, "vhost")
	return writeRedirectFiles(vhostDir, redirects)
}

// ========== PHPVhost ==========

func (v *PHPVhost) PHP() int {
	return v.parser.GetPHP()
}

func (v *PHPVhost) SetPHP(version int) error {
	// 先移除所有 PHP 相关的 include
	includes := v.Includes()
	var newIncludes []types.IncludeFile
	for _, inc := range includes {
		// 过滤掉 enable-php-*.conf
		if !strings.HasPrefix(inc.Path, "enable-php-") || !strings.HasSuffix(inc.Path, ".conf") {
			newIncludes = append(newIncludes, inc)
		}
	}

	// 如果版本不为 0，添加新的 PHP include
	if version > 0 {
		newIncludes = append(newIncludes, types.IncludeFile{
			Path:    fmt.Sprintf("enable-php-%d.conf", version),
			Comment: []string{fmt.Sprintf("# Enable PHP %d.%d", version/10, version%10)},
		})
	}

	return v.SetIncludes(newIncludes)
}

// ========== ProxyVhost ==========

func (v *ProxyVhost) Proxies() []types.Proxy {
	vhostDir := filepath.Join(v.configDir, "vhost")
	proxies, _ := parseProxyFiles(vhostDir)
	return proxies
}

func (v *ProxyVhost) SetProxies(proxies []types.Proxy) error {
	vhostDir := filepath.Join(v.configDir, "vhost")
	return writeProxyFiles(vhostDir, proxies)
}

func (v *ProxyVhost) ClearProxies() error {
	vhostDir := filepath.Join(v.configDir, "vhost")
	return clearProxyFiles(vhostDir)
}

func (v *ProxyVhost) Upstreams() map[string]types.Upstream {
	globalDir := filepath.Join(v.configDir, "global")
	upstreams, _ := parseUpstreamFiles(globalDir)
	return upstreams
}

func (v *ProxyVhost) SetUpstreams(upstreams map[string]types.Upstream) error {
	globalDir := filepath.Join(v.configDir, "global")
	return writeUpstreamFiles(globalDir, upstreams)
}

func (v *ProxyVhost) ClearUpstreams() error {
	globalDir := filepath.Join(v.configDir, "global")
	return clearUpstreamFiles(globalDir)
}
