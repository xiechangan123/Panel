package caddy

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
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

// baseVhost Caddy 虚拟主机基础实现
// 站点文件由内存状态整体生成，加载时解析回来；监听端口与域名编码在站点块地址里，绑定 IP 由 bind 指令表达
type baseVhost struct {
	configDir string
	siteDir   string
	siteName  string
	safeName  string

	root      string
	index     []string
	accessLog string
	includes  []types.IncludeFile
	listens   []types.Listen
	domains   []string
	ssl       *types.SSLConfig
	auths     []types.BasicAuth
	php       uint
	proxies   []types.Proxy
	upstreams []types.Upstream
	redirects []types.Redirect
}

// newBaseVhost 创建基础虚拟主机实例
func newBaseVhost(configDir string) (*baseVhost, error) {
	if configDir == "" {
		return nil, errors.New("config directory is required")
	}

	siteDir := filepath.Dir(configDir)
	siteName := filepath.Base(siteDir)
	v := &baseVhost{
		configDir: configDir,
		siteDir:   siteDir,
		siteName:  siteName,
		safeName:  safeName(siteName),
		root:      filepath.Join(siteDir, "public"),
		index:     []string{"index.html"},
		accessLog: filepath.Join(siteDir, "log", "access.log"),
		listens:   []types.Listen{{Address: "80", Args: []string{}}},
		domains:   []string{"localhost"},
	}

	path := filepath.Join(configDir, ConfName)
	if _, err := os.Stat(path); err == nil {
		cfg, err := ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse caddy config: %w", err)
		}
		v.load(cfg)
	}

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

// ========== 加载 ==========

var phpSocketPattern = regexp.MustCompile(`php-cgi-(\d+)\.sock$`)

func (v *baseVhost) load(cfg *conf.Config) {
	v.loadUpstreams(cfg)
	sites := sites(cfg)
	if len(sites) == 0 {
		return
	}
	v.loadListen(sites)
	v.loadSSL(sites)

	// 站点主体在片段里，手写的最简配置则直接取站点块
	body := sites[0].Block
	if snippet := cfg.Get("(" + v.siteSnippet() + ")"); snippet != nil {
		body = snippet.Block
	}
	if d := body.Get("root"); d != nil {
		v.root = d.Arg(len(d.Values()) - 1)
	}
	v.index = []string{}
	if fs := body.Get("file_server"); fs != nil {
		if idx := fs.Get("index"); idx != nil {
			v.index = idx.Values()
		}
	}
	v.accessLog = ""
	if l := body.Get("log"); l != nil {
		if out := l.Get("output"); out != nil && out.Arg(0) == "file" {
			v.accessLog = out.Arg(1)
		}
	}
	for _, d := range body.GetAll("import") {
		if !strings.HasPrefix(d.Arg(0), v.configDir+"/") {
			v.includes = append(v.includes, types.IncludeFile{Path: d.Arg(0)})
		}
	}
	if d := body.Get("php_fastcgi"); d != nil {
		if m := phpSocketPattern.FindStringSubmatch(d.Arg(0)); m != nil {
			version, _ := strconv.Atoi(m[1])
			v.php = uint(version)
		}
	}
	v.loadAuths(body)
	v.loadProxies(body)
	v.loadRedirects(body)
}

// loadListen 由各站点块地址还原监听与域名：无 scheme 时 80 端口为 HTTP，带主机名或块内有 tls 的其它端口为 HTTPS
func (v *baseVhost) loadListen(sites []*conf.Directive) {
	var hosts, ports, binds []string
	ssl := map[string]bool{}
	for _, site := range sites {
		hasTLS := site.Get("tls") != nil
		for _, key := range addresses(site) {
			scheme, rest, found := strings.Cut(key, "://")
			if !found {
				scheme, rest = "", key
			}
			host, port := splitHostPort(rest)
			if port == "" {
				port = map[bool]string{true: "80", false: "443"}[scheme == "http"]
			}
			secure := scheme == "https" || (scheme == "" && port != "80" && (host != "" || hasTLS))
			if host != "" && !slices.Contains(hosts, host) {
				hosts = append(hosts, host)
			}
			if !slices.Contains(ports, port) {
				ports = append(ports, port)
			}
			ssl[port] = ssl[port] || secure
		}
		if d := site.Get("bind"); d != nil {
			for _, bind := range d.Values() {
				if !slices.Contains(binds, bind) {
					binds = append(binds, bind)
				}
			}
		}
	}
	if hosts == nil {
		hosts = []string{}
	}
	v.domains = hosts

	if len(binds) == 0 {
		binds = []string{""}
	}
	v.listens = []types.Listen{}
	for _, port := range ports {
		args := []string{}
		if ssl[port] {
			args = []string{"ssl"}
		}
		for _, bind := range binds {
			address := port
			if bind != "" {
				address = net.JoinHostPort(bind, port)
			}
			v.listens = append(v.listens, types.Listen{Address: address, Args: slices.Clone(args)})
		}
	}
}

func (v *baseVhost) loadSSL(sites []*conf.Directive) {
	for _, site := range sites {
		tls := site.Get("tls")
		if tls == nil {
			continue
		}
		v.ssl = &types.SSLConfig{
			Cert:      tls.Arg(0),
			Key:       tls.Arg(1),
			Protocols: protocolsFromCaddy(nil),
			OCSP:      true,
		}
		if p := tls.Get("protocols"); p != nil {
			v.ssl.Protocols = protocolsFromCaddy(p.Values())
		}
		for _, d := range site.GetAll("header") {
			if slices.Contains(d.Values(), hstsHeader) {
				v.ssl.HSTS = true
			}
		}
	}
	if v.ssl == nil {
		return
	}
	for _, site := range sites {
		for _, d := range site.GetAll("redir") {
			if d.Arg(0) == httpMatcher {
				v.ssl.HTTPRedirect = true
			}
		}
	}
}

func (v *baseVhost) loadAuths(body *conf.Block) {
	for _, d := range body.GetAll("basic_auth") {
		m := body.Get(d.Arg(0))
		if m == nil {
			continue
		}
		pattern := m.Arg(1)
		if m.Block != nil {
			if p := m.Get("path"); p != nil {
				pattern = p.Arg(0)
			}
		}
		imp := d.Get("import")
		if imp == nil {
			continue
		}
		v.auths = append(v.auths, types.BasicAuth{
			Path:     "/" + strings.Trim(strings.TrimSuffix(pattern, "*"), "/"),
			UserFile: strings.TrimSuffix(imp.Arg(0), ".caddy"),
		})
	}
}

// ========== 核心方法 ==========

func (v *baseVhost) Enable() bool {
	_, err := os.Stat(filepath.Join(v.configDir, "site", "00-disable.conf"))
	return os.IsNotExist(err)
}

func (v *baseVhost) SetEnable(enable bool) error {
	disableConf := filepath.Join(v.configDir, "site", "00-disable.conf")
	if enable {
		if err := os.Remove(disableConf); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove disable config: %w", err)
		}
		return nil
	}
	// 标记文件，停用规则在生成站点块时按标记写入；文件本身只含注释，被 import 时无副作用
	if err := os.WriteFile(disableConf, []byte("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n"), 0600); err != nil {
		return fmt.Errorf("failed to write disable config: %w", err)
	}
	return nil
}

func (v *baseVhost) Listen() []types.Listen {
	return v.listens
}

// SetListen 绑定 IP 对整个站点块生效，监听地址要么都带 IP 要么都不带
func (v *baseVhost) SetListen(listens []types.Listen) error {
	withHost, withoutHost := 0, 0
	for _, l := range listens {
		if host, _ := splitHostPort(l.Address); host != "" {
			withHost++
		} else {
			withoutHost++
		}
	}
	if withHost > 0 && withoutHost > 0 {
		return errors.New("caddy binds addresses per site: listens must either all specify an IP or none")
	}

	v.listens = make([]types.Listen, 0, len(listens))
	for _, l := range listens {
		args := slices.Clone(l.Args)
		if args == nil {
			args = []string{}
		}
		v.listens = append(v.listens, types.Listen{Address: l.Address, Args: args})
	}
	return nil
}

func (v *baseVhost) ServerName() []string {
	return v.domains
}

func (v *baseVhost) SetServerName(serverName []string) error {
	if len(serverName) == 0 {
		return nil
	}
	v.domains = slices.Clone(serverName)
	return nil
}

func (v *baseVhost) Index() []string {
	if v.index == nil {
		return []string{}
	}
	return v.index
}

func (v *baseVhost) SetIndex(index []string) error {
	v.index = slices.Clone(index)
	return nil
}

func (v *baseVhost) Root() string {
	return v.root
}

func (v *baseVhost) SetRoot(root string) error {
	v.root = root
	return nil
}

func (v *baseVhost) Includes() []types.IncludeFile {
	return v.includes
}

func (v *baseVhost) SetIncludes(includes []types.IncludeFile) error {
	v.includes = slices.Clone(includes)
	return nil
}

func (v *baseVhost) AccessLog() string {
	return v.accessLog
}

func (v *baseVhost) SetAccessLog(accessLog string) error {
	v.accessLog = accessLog
	return nil
}

// ErrorLog Caddy 没有站点级错误日志，统一指向全局错误日志
func (v *baseVhost) ErrorLog() string {
	return ErrorLogPath
}

func (v *baseVhost) SetErrorLog(string) error {
	return nil
}

func (v *baseVhost) Save() error {
	users, err := v.writeUserFiles()
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(v.configDir, ConfName), []byte(Export(v.build(users))), 0600); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}
	return nil
}

// caddyUserFile 供 basic_auth import 的用户文件，与面板可回读的明文 htpasswd 并存
func caddyUserFile(userFile string) string {
	return userFile + ".caddy"
}

// writeUserFiles 把明文 htpasswd 转成 `user hash` 行，Caddy 只接受 bcrypt 或 argon2id；返回各文件是否有用户
func (v *baseVhost) writeUserFiles() (map[string]bool, error) {
	users := make(map[string]bool, len(v.auths))
	for _, auth := range v.auths {
		content, err := os.ReadFile(auth.UserFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read htpasswd file: %w", err)
		}
		var lines []string
		for line := range strings.SplitSeq(string(content), "\n") {
			user, password, ok := strings.Cut(strings.TrimSpace(line), ":")
			if !ok || user == "" || strings.HasPrefix(user, "#") {
				continue
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimPrefix(password, "{PLAIN}")), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			lines = append(lines, user+" "+string(hash))
		}
		if err = os.WriteFile(caddyUserFile(auth.UserFile), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
			return nil, fmt.Errorf("failed to write user file: %w", err)
		}
		users[auth.UserFile] = len(lines) > 0
	}
	return users, nil
}

func (v *baseVhost) Reset() error {
	*v = baseVhost{
		configDir: v.configDir,
		siteDir:   v.siteDir,
		siteName:  v.siteName,
		safeName:  v.safeName,
		root:      filepath.Join(v.siteDir, "public"),
		index:     []string{"index.html"},
		accessLog: filepath.Join(v.siteDir, "log", "access.log"),
		listens:   []types.Listen{{Address: "80", Args: []string{}}},
		domains:   []string{"localhost"},
	}
	return nil
}

func (v *baseVhost) Config(name string, scope types.ConfigScope) string {
	content, err := os.ReadFile(filepath.Join(v.configDir, string(scope), name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func (v *baseVhost) SetConfig(name string, scope types.ConfigScope, content string) error {
	return v.writeConfig(name, scope, "# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n"+content)
}

func (v *baseVhost) SetRawConfig(name string, scope types.ConfigScope, content string) error {
	return v.writeConfig(name, scope, content)
}

func (v *baseVhost) writeConfig(name string, scope types.ConfigScope, content string) error {
	if err := os.WriteFile(filepath.Join(v.configDir, string(scope), name), []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func (v *baseVhost) RemoveConfig(name string, scope types.ConfigScope) error {
	if err := os.Remove(filepath.Join(v.configDir, string(scope), name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}
	return nil
}

// ========== SSL ==========

func (v *baseVhost) SSL() bool {
	return v.ssl != nil
}

func (v *baseVhost) SSLConfig() *types.SSLConfig {
	if v.ssl == nil {
		return nil
	}
	cfg := *v.ssl
	return &cfg
}

func (v *baseVhost) SetSSLConfig(cfg *types.SSLConfig) error {
	if cfg == nil {
		return errors.New("SSL config cannot be nil")
	}
	ssl := *cfg
	if len(ssl.Protocols) == 0 {
		ssl.Protocols = []string{"TLSv1.2", "TLSv1.3"}
	}
	v.ssl = &ssl
	return nil
}

func (v *baseVhost) ClearSSL() error {
	v.ssl = nil
	return nil
}

// ========== 高级功能 ==========

// RateLimit Caddy 核心没有并发与带宽限制，站点级不支持
func (v *baseVhost) RateLimit() *types.RateLimit {
	return nil
}

func (v *baseVhost) SetRateLimit(*types.RateLimit) error {
	return nil
}

func (v *baseVhost) ClearRateLimit() error {
	return nil
}

func (v *baseVhost) BasicAuth() []types.BasicAuth {
	return v.auths
}

func (v *baseVhost) SetBasicAuth(auths []types.BasicAuth) error {
	v.auths = slices.Clone(auths)
	return nil
}

func (v *baseVhost) ClearBasicAuth() error {
	v.auths = nil
	return nil
}

// RealIP 真实 IP 只能在全局 servers 选项里配置可信代理，站点级不支持
func (v *baseVhost) RealIP() *types.RealIP {
	return nil
}

func (v *baseVhost) SetRealIP(*types.RealIP) error {
	return nil
}

func (v *baseVhost) ClearRealIP() error {
	return nil
}

func (v *baseVhost) Redirects() []types.Redirect {
	return v.redirects
}

func (v *baseVhost) SetRedirects(redirects []types.Redirect) error {
	v.redirects = slices.Clone(redirects)
	return nil
}

// ========== PHPVhost ==========

func (v *PHPVhost) PHP() uint {
	return v.php
}

func (v *PHPVhost) SetPHP(version uint) error {
	v.php = version
	return nil
}

// ========== ProxyVhost ==========

func (v *ProxyVhost) Proxies() []types.Proxy {
	return v.proxies
}

func (v *ProxyVhost) SetProxies(proxies []types.Proxy) error {
	v.proxies = slices.Clone(proxies)
	return nil
}

func (v *ProxyVhost) ClearProxies() error {
	v.proxies = nil
	return nil
}

func (v *ProxyVhost) Upstreams() []types.Upstream {
	return v.upstreams
}

func (v *ProxyVhost) SetUpstreams(upstreams []types.Upstream) error {
	v.upstreams = slices.Clone(upstreams)
	return nil
}

func (v *ProxyVhost) ClearUpstreams() error {
	v.upstreams = nil
	return nil
}

// ========== 生成 ==========

const (
	hstsHeader  = "Strict-Transport-Security"
	httpMatcher = "@ace_http"
	acmeMatcher = "@ace_acme"
	stopMatcher = "@ace_stop"
)

// build 由内存状态生成完整站点文件：主体写成片段，HTTP 与 HTTPS 地址各占一个站点块引用它，
// 避免同一站点块混用两种 scheme 时被合并进同一个 server；块内指令的执行顺序由 Caddy 按指令类型排定
func (v *baseVhost) build(users map[string]bool) *conf.Config {
	cfg := &conf.Config{}
	cfg.Append(conf.Cmt("Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"))
	// 共享片段目录通常为空，空通配会让 Caddy 每次重载都记一条警告，有文件时才引用
	if matches, _ := filepath.Glob(filepath.Join(v.configDir, "shared", "*.conf")); len(matches) > 0 {
		cfg.Add("import", filepath.Join(v.configDir, "shared", "*.conf"))
	}
	v.buildUpstreams(cfg)

	body := cfg.AddBlock("(" + v.siteSnippet() + ")")
	if v.accessLog != "" {
		l := body.AddBlock("log")
		l.Add("output", "file", v.accessLog)
		f := l.AddBlock("format", "transform", accessLogFormat)
		f.Add("time_format", accessLogTimeFormat)
	}
	body.Add("root", "*", v.root)
	body.Add("encode", "br", "zstd", "gzip")

	// 验证请求按原始 URI 匹配，不受伪静态改写影响；token 目录以 /.well-known/acme-challenge/ 为子路径，
	// 根目录只给 file_server 用，不改写请求的 root 变量，错误页仍按站点根目录解析
	body.Append(&conf.Blank{})
	body.Add(acmeMatcher, "expression", `{http.request.orig_uri.path}.startsWith("`+acmeURI+`")`)
	acme := body.AddBlock("handle", acmeMatcher)
	acme.Add("rewrite", "*", "{http.request.orig_uri.path}")
	acme.AddBlock("file_server").Add("root", ACMEDir)

	// 停用时恒真匹配器排在片段之前，先于用户的 handle 命中
	if !v.Enable() {
		body.Append(&conf.Comment{Text: " ace:stop"})
		body.Add(stopMatcher, "expression", "true")
		stop := body.AddBlock("handle", stopMatcher)
		stop.Add("rewrite", "*", "/stop.html")
		stop.AddBlock("file_server").Add("root", HTMLDir)
	}

	body.Append(&conf.Blank{})
	body.Add("import", filepath.Join(v.configDir, "site", "*.conf"))
	for _, inc := range v.includes {
		body.Add("import", inc.Path)
	}

	v.buildRedirects(body.Block)
	v.buildAuths(body.Block, users)
	v.buildProxies(body.Block)

	body.Append(&conf.Blank{})
	if v.php > 0 {
		body.Add("php_fastcgi", phpSocket(v.php))
	}
	if len(v.index) > 0 {
		body.AddBlock("file_server").Add("index", v.index...)
	} else {
		body.Add("file_server")
	}

	httpKeys, httpsKeys := v.siteKeys()
	binds := v.bindHosts()
	if len(httpKeys) > 0 {
		site := addSite(cfg, httpKeys...)
		if len(binds) > 0 {
			site.Add("bind", binds...)
		}
		if v.ssl != nil && v.ssl.HTTPRedirect && len(httpsKeys) > 0 {
			site.Add(httpMatcher, "not", "path", acmeURI+"*")
			site.Add("redir", httpMatcher, "https://{host}{uri}", "301")
		}
		site.Add("import", v.siteSnippet())
	}
	if len(httpsKeys) > 0 {
		site := addSite(cfg, httpsKeys...)
		if len(binds) > 0 {
			site.Add("bind", binds...)
		}
		v.buildTLS(site)
		if v.ssl != nil && v.ssl.HSTS {
			site.Add("header", hstsHeader, "max-age=31536000")
		}
		site.Add("import", v.siteSnippet())
	}

	return cfg
}

// siteSnippet 站点主体片段名
func (v *baseVhost) siteSnippet() string {
	return "ace_site_" + v.safeName
}

// siteKeys 每个监听端口与每个域名组合成一个显式带 scheme 的地址，按 scheme 分为两组；无域名时为无主机名地址
func (v *baseVhost) siteKeys() ([]string, []string) {
	var httpKeys, httpsKeys []string
	for _, l := range v.listens {
		_, port := splitHostPort(l.Address)
		if port == "" {
			port = l.Address
		}
		keys, scheme := &httpKeys, "http"
		if slices.Contains(l.Args, "ssl") {
			keys, scheme = &httpsKeys, "https"
		}
		if len(v.domains) == 0 {
			*keys = append(*keys, scheme+"://:"+port)
			continue
		}
		for _, host := range v.domains {
			*keys = append(*keys, scheme+"://"+host+":"+port)
		}
	}
	return slices.Compact(httpKeys), slices.Compact(httpsKeys)
}

// bindHosts 监听地址里出现的 IP，去重后作为 bind 参数
func (v *baseVhost) bindHosts() []string {
	var hosts []string
	for _, l := range v.listens {
		if host, _ := splitHostPort(l.Address); host != "" && !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

func (v *baseVhost) buildTLS(site *conf.Directive) {
	if v.ssl == nil {
		return
	}
	protocols := protocolsToCaddy(v.ssl.Protocols)
	if len(protocols) == 0 {
		site.Add("tls", v.ssl.Cert, v.ssl.Key)
		return
	}
	site.AddBlock("tls", v.ssl.Cert, v.ssl.Key).Add("protocols", protocols...)
}

func (v *baseVhost) authMatcher(i int) string {
	return fmt.Sprintf("@ace_auth_%d", i)
}

// buildAuths 每条认证一个路径匹配器，整站认证需放过验证路径；用户文件按 `user hash` 行直接 import 进块内，没有用户的规则不生成
func (v *baseVhost) buildAuths(body *conf.Block, users map[string]bool) {
	for i, auth := range v.auths {
		if !users[auth.UserFile] {
			continue
		}
		name := v.authMatcher(i)
		pattern := "/" + strings.Trim(auth.Path, "/") + "*"
		if auth.Path == "/" {
			pattern = "/*"
			m := body.AddBlock(name)
			m.Add("path", pattern)
			m.Add("not", "path", acmeURI+"*")
		} else {
			body.Add(name, "path", pattern)
		}
		body.AddBlock("basic_auth", name).Add("import", caddyUserFile(auth.UserFile))
	}
}

// ========== 辅助 ==========

// accessLogFormat nginx combined 格式，由 transform-encoder 插件渲染，空字段输出 -
const accessLogFormat = `{request>remote_ip} - {user_id} [{ts}] "{request>method} {request>uri} {request>proto}" {status} {size} "{request>headers>Referer>[0]}" "{request>headers>User-Agent>[0]}"`

const accessLogTimeFormat = "02/Jan/2006:15:04:05 -0700"

// phpSocket 面板 PHP-FPM 的套接字地址
func phpSocket(version uint) string {
	return fmt.Sprintf("unix//tmp/php-cgi-%d.sock", version)
}

// splitHostPort 拆分 `[ip:]port`，无法拆分时整体视为端口
func splitHostPort(address string) (string, string) {
	if !strings.Contains(address, ":") {
		return "", address
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", address
	}
	return host, port
}

// safeName 将名称转换为 Caddyfile 标识符安全形式；片段名全局唯一，编码必须可逆，避免 a-b 与 a_b 撞名
func safeName(name string) string {
	var b strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9':
			b.WriteRune(char)
		case char == '_':
			b.WriteString("__")
		default:
			_, _ = fmt.Fprintf(&b, "_%02x", char)
		}
	}
	return b.String()
}

var caddyProtocols = map[string]string{"TLSv1.2": "tls1.2", "TLSv1.3": "tls1.3"}

// protocolsToCaddy 转为 `protocols <min> [<max>]`，Caddy 只支持 TLS 1.2 与 1.3，其余版本忽略；两者都选时省略，与默认一致
func protocolsToCaddy(protocols []string) []string {
	has12, has13 := false, false
	for _, p := range protocols {
		switch caddyProtocols[p] {
		case "tls1.2":
			has12 = true
		case "tls1.3":
			has13 = true
		}
	}
	switch {
	case has12 && !has13:
		return []string{"tls1.2", "tls1.2"}
	case has13 && !has12:
		return []string{"tls1.3"}
	}
	return nil
}

func protocolsFromCaddy(args []string) []string {
	switch {
	case len(args) == 0:
		return []string{"TLSv1.2", "TLSv1.3"}
	case len(args) == 2 && args[0] == "tls1.2" && args[1] == "tls1.2":
		return []string{"TLSv1.2"}
	case args[0] == "tls1.3":
		return []string{"TLSv1.3"}
	}
	return []string{"TLSv1.2", "TLSv1.3"}
}
