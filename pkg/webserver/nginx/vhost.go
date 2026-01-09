package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tufanbarisyildirim/gonginx/config"

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
	directive, err := v.parser.FindOne("server.root")
	if err != nil {
		return false
	}

	return directive.GetParameters()[0].GetValue() != DisablePagePath
}

func (v *baseVhost) SetEnable(enable bool) error {
	path := DisablePagePath

	if enable {
		// 尝试获取保存的根目录
		if root, err := os.ReadFile(filepath.Join(v.configDir, "root.saved")); err != nil {
			path = filepath.Join(SitesPath, filepath.Dir(v.configDir), "public") // 默认根目录
		} else {
			path = strings.TrimSpace(string(root))
		}
	} else {
		// 禁用时，保存当前根目录
		currentRoot := v.Root()
		if currentRoot != "" && currentRoot != DisablePagePath {
			if err := os.WriteFile(filepath.Join(v.configDir, "root.saved"), []byte(currentRoot), 0644); err != nil {
				return fmt.Errorf("failed to save current root: %w", err)
			}
		}
	}

	// 设置根目录
	_ = v.parser.Clear("server.root")
	if err := v.SetRoot(path); err != nil {
		return err
	}

	// 清理保存的根目录文件
	if enable {
		_ = os.RemoveAll(filepath.Join(v.configDir, "root.saved"))
	}

	// 设置导入配置
	_ = v.parser.Clear("server.include")
	if enable {
		return v.parser.Set("server", []*config.Directive{
			{
				Name:       "include",
				Parameters: v.parser.slices2Parameters([]string{fmt.Sprintf("%s/site/*.conf", v.configDir)}),
			},
		})
	}

	return nil
}

func (v *baseVhost) Listen() []types.Listen {
	directives, err := v.parser.Find("server.listen")
	if err != nil {
		return nil
	}

	var result []types.Listen
	for _, dir := range directives {
		l := v.parser.parameters2Slices(dir.GetParameters())
		listen := types.Listen{Address: l[0], Args: []string{}}
		for i := 1; i < len(l); i++ {
			listen.Args = append(listen.Args, l[i])
		}

		result = append(result, listen)
	}

	return result
}

func (v *baseVhost) SetListen(listens []types.Listen) error {
	var directives []*config.Directive
	for _, l := range listens {
		listen := []string{l.Address}
		listen = append(listen, l.Args...)
		directives = append(directives, &config.Directive{
			Name:       "listen",
			Parameters: v.parser.slices2Parameters(listen),
		})
	}

	_ = v.parser.Clear("server.listen")

	return v.parser.Set("server", directives)
}

func (v *baseVhost) ServerName() []string {
	directive, err := v.parser.FindOne("server.server_name")
	if err != nil {
		return nil
	}

	return v.parser.parameters2Slices(directive.GetParameters())
}

func (v *baseVhost) SetServerName(serverName []string) error {
	_ = v.parser.Clear("server.server_name")

	return v.parser.Set("server", []*config.Directive{
		{
			Name:       "server_name",
			Parameters: v.parser.slices2Parameters(serverName),
		},
	})
}

func (v *baseVhost) Index() []string {
	directive, err := v.parser.FindOne("server.index")
	if err != nil {
		return nil
	}

	return v.parser.parameters2Slices(directive.GetParameters())
}

func (v *baseVhost) SetIndex(index []string) error {
	_ = v.parser.Clear("server.index")

	return v.parser.Set("server", []*config.Directive{
		{
			Name:       "index",
			Parameters: v.parser.slices2Parameters(index),
		},
	})
}

func (v *baseVhost) Root() string {
	directive, err := v.parser.FindOne("server.root")
	if err != nil {
		return ""
	}
	if len(v.parser.parameters2Slices(directive.GetParameters())) == 0 {
		return ""
	}

	return directive.GetParameters()[0].GetValue()
}

func (v *baseVhost) SetRoot(root string) error {
	_ = v.parser.Clear("server.root")

	return v.parser.Set("server", []*config.Directive{
		{
			Name:       "root",
			Parameters: []config.Parameter{{Value: root}},
		},
	})
}

func (v *baseVhost) Includes() []types.IncludeFile {
	directives, err := v.parser.Find("server.include")
	if err != nil {
		return nil
	}

	var result []types.IncludeFile

	for _, dir := range directives {
		if len(dir.GetParameters()) != 1 {
			return nil
		}
		result = append(result, types.IncludeFile{
			Path:    dir.GetParameters()[0].GetValue(),
			Comment: dir.GetComment(),
		})
	}

	return result
}

func (v *baseVhost) SetIncludes(includes []types.IncludeFile) error {
	_ = v.parser.Clear("server.include")

	var directives []*config.Directive
	for _, inc := range includes {
		directives = append(directives, &config.Directive{
			Name:       "include",
			Parameters: []config.Parameter{{Value: inc.Path}},
			Comment:    inc.Comment,
		})
	}

	return v.parser.Set("server", directives)
}

func (v *baseVhost) AccessLog() string {
	content := v.Config("020-access-log.conf", "site")
	if content == "" {
		return ""
	}

	var result string
	_, err := fmt.Sscanf(content, "access_log %s", &result)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(result, ";")
}

func (v *baseVhost) SetAccessLog(accessLog string) error {
	if accessLog == "" {
		return v.RemoveConfig("020-access-log.conf", "site")
	}
	return v.SetConfig("020-access-log.conf", "site", fmt.Sprintf("access_log %s;\n", accessLog))
}

func (v *baseVhost) ErrorLog() string {
	content := v.Config("020-error-log.conf", "site")
	if content == "" {
		return ""
	}

	var result string
	_, err := fmt.Sscanf(content, "error_log %s", &result)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(result, ";")
}

func (v *baseVhost) SetErrorLog(errorLog string) error {
	if errorLog == "" {
		return v.RemoveConfig("020-error-log.conf", "site")
	}
	return v.SetConfig("020-error-log.conf", "site", fmt.Sprintf("error_log %s;\n", errorLog))
}

func (v *baseVhost) Save() error {
	return v.parser.Save()
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

func (v *baseVhost) Config(name string, typ string) string {
	conf := filepath.Join(v.configDir, typ, name)
	content, err := os.ReadFile(conf)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func (v *baseVhost) SetConfig(name string, typ string, content string) error {
	conf := filepath.Join(v.configDir, typ, name)
	if err := os.WriteFile(conf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func (v *baseVhost) RemoveConfig(name string, typ string) error {
	conf := filepath.Join(v.configDir, typ, name)
	if err := os.Remove(conf); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}
	return nil
}

func (v *baseVhost) SSL() bool {
	directive, err := v.parser.FindOne("server.ssl_certificate")
	if err != nil {
		return false
	}
	if len(v.parser.parameters2Slices(directive.GetParameters())) == 0 {
		return false
	}

	return true
}

func (v *baseVhost) SSLConfig() *types.SSLConfig {
	if !v.SSL() {
		return nil
	}

	protocols, _ := v.parser.FindOne("server.ssl_protocols")
	ciphers, _ := v.parser.FindOne("server.ssl_ciphers")
	hsts := false
	ocsp := false
	httpRedirect := false
	altSvc := ""

	directives, _ := v.parser.Find("server.add_header")
	for _, dir := range directives {
		if slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "Strict-Transport-Security") {
			hsts = true
			break
		}
	}
	directive, err := v.parser.FindOne("server.ssl_stapling")
	if err == nil {
		if len(v.parser.parameters2Slices(directive.GetParameters())) != 0 {
			ocsp = directive.GetParameters()[0].GetValue() == "on"
		}
	}
	directives, _ = v.parser.Find("server.if")
	for _, dir := range directives {
		for _, dir2 := range dir.GetBlock().GetDirectives() {
			if dir2.GetName() == "return" && slices.Contains(v.parser.parameters2Slices(dir2.GetParameters()), "https://$host$request_uri") {
				httpRedirect = true
				break
			}
		}
	}
	directive, err = v.parser.FindOne("server.add_header")
	if err == nil {
		for i, param := range v.parser.parameters2Slices(directive.GetParameters()) {
			if strings.HasPrefix(param, "Alt-Svc") && i+1 < len(v.parser.parameters2Slices(directive.GetParameters())) {
				altSvc = v.parser.parameters2Slices(directive.GetParameters())[i+1]
				break
			}
		}
	}

	return &types.SSLConfig{
		Protocols:    v.parser.parameters2Slices(protocols.GetParameters()),
		Ciphers:      ciphers.GetParameters()[0].GetValue(),
		HSTS:         hsts,
		OCSP:         ocsp,
		HTTPRedirect: httpRedirect,
		AltSvc:       altSvc,
	}
}

func (v *baseVhost) SetSSLConfig(cfg *types.SSLConfig) error {
	if cfg == nil {
		return fmt.Errorf("SSL config cannot be nil")
	}

	if err := v.ClearSSL(); err != nil {
		return err
	}
	if len(cfg.Protocols) == 0 {
		cfg.Protocols = []string{"TLSv1.2", "TLSv1.3"}
	}
	if cfg.Ciphers == "" {
		cfg.Ciphers = "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305"
	}

	err := v.parser.Set("server", []*config.Directive{
		{
			Name:       "ssl_certificate",
			Parameters: []config.Parameter{{Value: cfg.Cert}},
		},
		{
			Name:       "ssl_certificate_key",
			Parameters: []config.Parameter{{Value: cfg.Key}},
		},
		{
			Name:       "ssl_session_timeout",
			Parameters: []config.Parameter{{Value: "1d"}},
		},
		{
			Name:       "ssl_session_cache",
			Parameters: []config.Parameter{{Value: "shared:SSL:10m"}},
		},
		{
			Name:       "ssl_protocols",
			Parameters: v.parser.slices2Parameters(cfg.Protocols),
		},
		{
			Name:       "ssl_ciphers",
			Parameters: []config.Parameter{{Value: cfg.Ciphers}},
		},
		{
			Name:       "ssl_prefer_server_ciphers",
			Parameters: []config.Parameter{{Value: "off"}},
		},
		{
			Name:       "ssl_early_data",
			Parameters: []config.Parameter{{Value: "on"}},
		},
	}, "root")
	if err != nil {
		return err
	}

	// 设置 HSTS
	if err = v.setHSTS(cfg.HSTS); err != nil {
		return err
	}

	// 设置 OCSP
	if cfg.OCSP {
		if err = v.parser.Set("server", []*config.Directive{
			{
				Name:       "ssl_stapling",
				Parameters: []config.Parameter{{Value: "on"}},
			},
			{
				Name:       "ssl_stapling_verify",
				Parameters: []config.Parameter{{Value: "on"}},
			},
		}); err != nil {
			return err
		}
	}

	// 设置 HTTP 跳转
	if err = v.setHTTPSRedirect(cfg.HTTPRedirect); err != nil {
		return err
	}

	// 设置 Alt-Svc
	if err = v.setAltSvc(cfg.AltSvc); err != nil {
		return err
	}

	return nil
}

func (v *baseVhost) ClearSSL() error {
	_ = v.parser.Clear("server.ssl_certificate")
	_ = v.parser.Clear("server.ssl_certificate_key")
	_ = v.parser.Clear("server.ssl_session_timeout")
	_ = v.parser.Clear("server.ssl_session_cache")
	_ = v.parser.Clear("server.ssl_protocols")
	_ = v.parser.Clear("server.ssl_ciphers")
	_ = v.parser.Clear("server.ssl_prefer_server_ciphers")
	_ = v.parser.Clear("server.ssl_early_data")
	_ = v.parser.Clear("server.ssl_stapling")
	_ = v.parser.Clear("server.ssl_stapling_verify")
	_ = v.setHSTS(false)
	_ = v.setHTTPSRedirect(false)
	_ = v.setAltSvc("")

	return nil
}

func (v *baseVhost) RateLimit() *types.RateLimit {
	rate := ""
	directive, err := v.parser.FindOne("server.limit_rate")
	if err == nil {
		if len(v.parser.parameters2Slices(directive.GetParameters())) != 0 {
			rate = directive.GetParameters()[0].GetValue()
		}
	}
	directives, _ := v.parser.Find("server.limit_conn")
	var limitConn [][]string
	for _, dir := range directives {
		limitConn = append(limitConn, v.parser.parameters2Slices(dir.GetParameters()))
	}

	if rate == "" && len(limitConn) == 0 {
		return nil
	}

	rateLimit := &types.RateLimit{
		Rate: rate,
		Zone: make(map[string]string),
	}

	// 解析 limit_conn 配置
	for _, limit := range limitConn {
		if len(limit) >= 2 {
			// limit_conn zone connections
			// 例如: limit_conn perip 10
			rateLimit.Zone[limit[0]] = limit[1]
		}
	}

	return rateLimit
}

func (v *baseVhost) SetRateLimit(limit *types.RateLimit) error {
	var limitConns [][]string
	for zone, connections := range limit.Zone {
		limitConns = append(limitConns, []string{zone, connections})
	}

	// 设置限速
	_ = v.parser.Clear("server.limit_rate")
	if err := v.parser.Set("server", []*config.Directive{
		{
			Name:       "limit_rate",
			Parameters: []config.Parameter{{Value: limit.Rate}},
		},
	}); err != nil {
		return err
	}

	// 设置并发连接数限制
	_ = v.parser.Clear("server.limit_conn")
	var directives []*config.Directive
	for _, lim := range limitConns {
		if len(lim) >= 2 {
			directives = append(directives, &config.Directive{
				Name:       "limit_conn",
				Parameters: v.parser.slices2Parameters(lim),
			})
		}
	}

	return v.parser.Set("server", directives)
}

func (v *baseVhost) ClearRateLimit() error {
	_ = v.parser.Clear("server.limit_rate")
	_ = v.parser.Clear("server.limit_conn")
	return nil
}

func (v *baseVhost) BasicAuth() map[string]string {
	// auth_basic "realm"
	realmDir, err := v.parser.FindOne("server.auth_basic")
	if err != nil {
		return nil
	}

	// auth_basic_user_file /path/to/file
	fileDir, err := v.parser.FindOne("server.auth_basic_user_file")
	if err != nil {
		return nil
	}

	realm := ""
	if len(realmDir.GetParameters()) > 0 {
		realm = realmDir.GetParameters()[0].GetValue()
	}

	file := ""
	if len(fileDir.GetParameters()) > 0 {
		file = fileDir.GetParameters()[0].GetValue()
	}

	return map[string]string{
		"realm":     realm,
		"user_file": file,
	}
}

func (v *baseVhost) SetBasicAuth(auth map[string]string) error {
	_ = v.parser.Clear("server.auth_basic")
	_ = v.parser.Clear("server.auth_basic_user_file")

	realm := auth["realm"]
	userFile := auth["user_file"]

	if realm == "" {
		realm = "Restricted"
	}

	return v.parser.Set("server", []*config.Directive{
		{
			Name:       "auth_basic",
			Parameters: []config.Parameter{{Value: realm}},
		},
		{
			Name:       "auth_basic_user_file",
			Parameters: []config.Parameter{{Value: userFile}},
		},
	})
}

func (v *baseVhost) ClearBasicAuth() error {
	_ = v.parser.Clear("server.auth_basic")
	_ = v.parser.Clear("server.auth_basic_user_file")
	return nil
}

func (v *baseVhost) Redirects() []types.Redirect {
	siteDir := filepath.Join(v.configDir, "site")
	redirects, _ := parseRedirectFiles(siteDir)
	return redirects
}

func (v *baseVhost) SetRedirects(redirects []types.Redirect) error {
	siteDir := filepath.Join(v.configDir, "site")
	return writeRedirectFiles(siteDir, redirects)
}

// ========== PHPVhost ==========

func (v *PHPVhost) PHP() uint {
	content := v.Config("010-php.conf", "site")
	if content == "" {
		return 0
	}

	var result uint
	_, err := fmt.Sscanf(content, "include enable-php-%d.conf;", &result)
	if err != nil {
		return 0
	}
	return result
}

func (v *PHPVhost) SetPHP(version uint) error {
	if version == 0 {
		return v.RemoveConfig("010-php.conf", "site")
	}
	return v.SetConfig("010-php.conf", "site", fmt.Sprintf("include enable-php-%d.conf;\n", version))
}

// ========== ProxyVhost ==========

func (v *ProxyVhost) Proxies() []types.Proxy {
	siteDir := filepath.Join(v.configDir, "site")
	proxies, _ := parseProxyFiles(siteDir)
	return proxies
}

func (v *ProxyVhost) SetProxies(proxies []types.Proxy) error {
	siteDir := filepath.Join(v.configDir, "site")
	return writeProxyFiles(siteDir, proxies)
}

func (v *ProxyVhost) ClearProxies() error {
	siteDir := filepath.Join(v.configDir, "site")
	return clearProxyFiles(siteDir)
}

func (v *ProxyVhost) Upstreams() []types.Upstream {
	sharedDir := filepath.Join(v.configDir, "shared")
	upstreams, _ := parseUpstreamFiles(sharedDir)
	return upstreams
}

func (v *ProxyVhost) SetUpstreams(upstreams []types.Upstream) error {
	sharedDir := filepath.Join(v.configDir, "shared")
	return writeUpstreamFiles(sharedDir, upstreams)
}

func (v *ProxyVhost) ClearUpstreams() error {
	sharedDir := filepath.Join(v.configDir, "shared")
	return clearUpstreamFiles(sharedDir)
}

func (v *baseVhost) setHSTS(hsts bool) error {
	old, err := v.parser.Find("server.add_header")
	if err != nil {
		return err
	}
	if err = v.parser.Clear("server.add_header"); err != nil {
		return err
	}
	var directives []*config.Directive
	var foundFlag bool
	for _, dir := range old {
		if slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "Strict-Transport-Security") {
			foundFlag = true
			if hsts {
				directives = append(directives, &config.Directive{
					Name:       dir.GetName(),
					Parameters: []config.Parameter{{Value: "Strict-Transport-Security"}, {Value: "max-age=31536000"}},
					Comment:    dir.GetComment(),
				})
			}
		} else {
			directives = append(directives, &config.Directive{
				Name:       dir.GetName(),
				Parameters: dir.GetParameters(),
				Comment:    dir.GetComment(),
			})
		}
	}

	if !foundFlag && hsts {
		directives = append(directives, &config.Directive{
			Name:       "add_header",
			Parameters: []config.Parameter{{Value: "Strict-Transport-Security"}, {Value: "max-age=31536000"}},
		})
	}

	if err = v.parser.Set("server", directives); err != nil {
		return err
	}

	return nil
}

func (v *baseVhost) setHTTPSRedirect(httpRedirect bool) error {
	// if 重定向
	ifs, err := v.parser.Find("server.if")
	if err != nil {
		return err
	}
	if err = v.parser.Clear("server.if"); err != nil {
		return err
	}

	var directives []*config.Directive
	var foundFlag bool
	for _, dir := range ifs { // 所有 if
		if !httpRedirect {
			if len(dir.GetParameters()) == 3 && dir.GetParameters()[0].GetValue() == "($scheme" && dir.GetParameters()[1].GetValue() == "=" && dir.GetParameters()[2].GetValue() == "http)" {
				continue
			}
		}
		var ifDirectives []config.IDirective
		for _, dir2 := range dir.GetBlock().GetDirectives() { // 每个 if 中所有指令
			if !httpRedirect {
				// 不启用http重定向，则判断并移除特定的return指令
				if dir2.GetName() != "return" && !slices.Contains(v.parser.parameters2Slices(dir2.GetParameters()), "https://$host$request_uri") {
					ifDirectives = append(ifDirectives, dir2)
				}
			} else {
				// 启用http重定向，需要检查防止重复添加
				if dir2.GetName() == "return" && slices.Contains(v.parser.parameters2Slices(dir2.GetParameters()), "https://$host$request_uri") {
					foundFlag = true
				}
				ifDirectives = append(ifDirectives, dir2)
			}
		}
		// 写回 if 指令
		if block, ok := dir.GetBlock().(*config.Block); ok {
			block.Directives = ifDirectives
		}
		directives = append(directives, &config.Directive{
			Block:      dir.GetBlock(),
			Name:       dir.GetName(),
			Parameters: dir.GetParameters(),
			Comment:    dir.GetComment(),
		})
	}

	if !foundFlag && httpRedirect {
		ifDir := &config.Directive{
			Name:       "if",
			Block:      &config.Block{},
			Parameters: []config.Parameter{{Value: "($scheme"}, {Value: "="}, {Value: "http)"}},
		}
		redirectDir := &config.Directive{
			Name:       "return",
			Parameters: []config.Parameter{{Value: "308"}, {Value: "https://$host$request_uri"}},
		}
		redirectDir.SetParent(ifDir.GetParent())
		ifBlock := ifDir.GetBlock().(*config.Block)
		ifBlock.Directives = append(ifBlock.Directives, redirectDir)
		directives = append(directives, ifDir)
	}

	if err = v.parser.Set("server", directives); err != nil {
		return err
	}

	// error_page 497 重定向
	directives = nil
	errorPages, err := v.parser.Find("server.error_page")
	if err != nil {
		return err
	}
	if err = v.parser.Clear("server.error_page"); err != nil {
		return err
	}
	var found497 bool
	for _, dir := range errorPages {
		if !httpRedirect {
			// 不启用https重定向，则判断并移除特定的return指令
			if !slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "497") && !slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "https://$host:$server_port$request_uri") {
				directives = append(directives, &config.Directive{
					Block:      dir.GetBlock(),
					Name:       dir.GetName(),
					Parameters: dir.GetParameters(),
					Comment:    dir.GetComment(),
				})
			}
		} else {
			// 启用https重定向，需要检查防止重复添加
			if slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "497") && slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "https://$host:$server_port$request_uri") {
				found497 = true
			}
			directives = append(directives, &config.Directive{
				Block:      dir.GetBlock(),
				Name:       dir.GetName(),
				Parameters: dir.GetParameters(),
				Comment:    dir.GetComment(),
			})
		}
	}

	if !found497 && httpRedirect {
		directives = append(directives, &config.Directive{
			Name:       "error_page",
			Parameters: []config.Parameter{{Value: "497"}, {Value: "=308"}, {Value: "https://$host:$server_port$request_uri"}},
		})
	}

	return v.parser.Set("server", directives)
}

func (v *baseVhost) setAltSvc(altSvc string) error {
	old, err := v.parser.Find("server.add_header")
	if err != nil {
		return err
	}
	if err = v.parser.Clear("server.add_header"); err != nil {
		return err
	}

	var directives []*config.Directive
	var foundFlag bool
	for _, dir := range old {
		if slices.Contains(v.parser.parameters2Slices(dir.GetParameters()), "Alt-Svc") {
			foundFlag = true
			if altSvc != "" { // 为空表示要删除
				directives = append(directives, &config.Directive{
					Name:       dir.GetName(),
					Parameters: []config.Parameter{{Value: "Alt-Svc"}, {Value: altSvc}},
					Comment:    dir.GetComment(),
				})
			}
		} else {
			directives = append(directives, &config.Directive{
				Name:       dir.GetName(),
				Parameters: dir.GetParameters(),
				Comment:    dir.GetComment(),
			})
		}
	}

	if !foundFlag && altSvc != "" {
		directives = append(directives, &config.Directive{
			Name:       "add_header",
			Parameters: []config.Parameter{{Value: "Alt-Svc"}, {Value: altSvc}},
		})
	}

	if err = v.parser.Set("server", directives); err != nil {
		return err
	}

	return nil
}
