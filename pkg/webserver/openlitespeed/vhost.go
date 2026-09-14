package openlitespeed

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

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

// baseVhost OpenLiteSpeed 虚拟主机基础实现
// vhconf 由面板从内存状态整体生成，加载时再从文件解析回状态；监听地址与域名属于主配置的 listener，单独记录在 listen 文件中
type baseVhost struct {
	configDir string
	siteDir   string
	siteName  string
	safeName  string

	root      string
	index     []string
	accessLog string
	errorLog  string
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
		errorLog:  filepath.Join(siteDir, "log", "error.log"),
		listens:   []types.Listen{{Address: "80", Args: []string{}}},
		domains:   []string{"localhost"},
	}

	cfg, err := parseIfExists(filepath.Join(configDir, VhostConfName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse openlitespeed config: %w", err)
	}
	if cfg != nil {
		v.load(cfg)
	}
	listen, err := parseIfExists(filepath.Join(configDir, ListenConfName))
	if err != nil {
		return nil, fmt.Errorf("failed to parse listen config: %w", err)
	}
	if listen != nil {
		v.loadListen(listen)
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

func (v *baseVhost) load(cfg *Config) {
	if root := cfg.Value("docRoot"); root != "" {
		v.root = root
	}
	if idx := cfg.Block("index"); idx != nil {
		v.index = splitList(idx.Value("indexFiles"))
	}
	v.accessLog = ""
	if b := cfg.Block("accesslog"); b != nil {
		v.accessLog = b.Arg
	}
	v.errorLog = ""
	if b := cfg.Block("errorlog"); b != nil {
		v.errorLog = b.Arg
	}
	for _, d := range cfg.Directives("include") {
		if !strings.HasPrefix(d.Value, v.configDir+"/") {
			v.includes = append(v.includes, types.IncludeFile{Path: d.Value})
		}
	}
	v.loadSSL(cfg)
	v.loadPHP(cfg)
	v.loadUpstreams(cfg)
	v.loadProxies(cfg)
	v.loadAuths(cfg)
	v.loadRedirects(cfg)
}

func (v *baseVhost) loadListen(cfg *Config) {
	v.listens = []types.Listen{}
	for _, d := range cfg.Directives("listen") {
		fields := strings.Fields(d.Value)
		if len(fields) == 0 {
			continue
		}
		args := fields[1:]
		if args == nil {
			args = []string{}
		}
		v.listens = append(v.listens, types.Listen{Address: fields[0], Args: args})
	}
	v.domains = strings.Fields(cfg.Value("domain"))
}

func (v *baseVhost) loadSSL(cfg *Config) {
	b := cfg.Block("vhssl")
	if b == nil {
		return
	}
	v.ssl = &types.SSLConfig{
		Cert:      b.Value("certFile"),
		Key:       b.Value("keyFile"),
		Protocols: protocolsFromBits(b.Value("sslProtocol")),
		OCSP:      b.Value("enableStapling") == "1",
	}
	for _, ctx := range cfg.Blocks("context") {
		if strings.Contains(ctx.Value("extraHeaders"), hstsHeader) {
			v.ssl.HSTS = true
		}
	}
	if rw := cfg.Block("rewrite"); rw != nil {
		for _, d := range rw.Directives("RewriteRule") {
			if strings.Contains(d.Value, "https://%{HTTP_HOST}") {
				v.ssl.HTTPRedirect = true
			}
		}
	}
}

func (v *baseVhost) loadPHP(cfg *Config) {
	sh := cfg.Block("scripthandler")
	if sh == nil {
		return
	}
	for _, d := range sh.Directives("include") {
		if m := phpHandlerPattern.FindStringSubmatch(d.Value); m != nil {
			version, _ := strconv.Atoi(m[1])
			v.php = uint(version)
		}
	}
}

func (v *baseVhost) loadAuths(cfg *Config) {
	for _, ctx := range cfg.Blocks("context") {
		realmName := ctx.Value("realm")
		if realmName == "" {
			continue
		}
		path := ctx.Meta("auth")
		if path == "" {
			path = "/" + strings.Trim(ctx.Arg, "/")
		}
		userFile := ""
		if realm := cfg.Block("realm", realmName); realm != nil {
			if db := realm.Block("userDB"); db != nil {
				userFile = db.Value("location")
			}
		}
		v.auths = append(v.auths, types.BasicAuth{Path: path, UserFile: userFile})
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
	// 标记文件，停用规则在生成 vhconf 时按标记写入
	if err := os.WriteFile(disableConf, []byte("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n"), 0600); err != nil {
		return fmt.Errorf("failed to write disable config: %w", err)
	}
	return nil
}

func (v *baseVhost) Listen() []types.Listen {
	return v.listens
}

func (v *baseVhost) SetListen(listens []types.Listen) error {
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

func (v *baseVhost) ErrorLog() string {
	return v.errorLog
}

func (v *baseVhost) SetErrorLog(errorLog string) error {
	v.errorLog = errorLog
	return nil
}

func (v *baseVhost) Save() error {
	if err := os.WriteFile(filepath.Join(v.configDir, VhostConfName), []byte(v.build().String()), 0600); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(v.configDir, ListenConfName), []byte(v.buildListen().String()), 0600); err != nil {
		return fmt.Errorf("failed to save listen file: %w", err)
	}

	return Sync()
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
		errorLog:  filepath.Join(v.siteDir, "log", "error.log"),
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

// RateLimit OpenLiteSpeed 的限速只能在服务器级配置，站点级不支持
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

// RealIP 真实 IP 只能在服务器级通过 useIpInProxyHeader 配置，站点级不支持
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

// build 由内存状态生成完整 vhconf
func (v *baseVhost) build() *Config {
	cfg := &Config{}
	cfg.Append(&Comment{Text: " Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"})
	cfg.Add("docRoot", v.root)

	idx := cfg.AddBlock("index", "")
	idx.Add("useServer", "0")
	if len(v.index) > 0 {
		idx.Add("indexFiles", strings.Join(v.index, ", "))
	}

	if v.errorLog != "" {
		b := cfg.AddBlock("errorlog", v.errorLog)
		b.Add("useServer", "0")
		b.Add("logLevel", "WARN")
		b.Add("rollingSize", "10M")
	}
	if v.accessLog != "" {
		b := cfg.AddBlock("accesslog", v.accessLog)
		b.Add("useServer", "0")
		b.Add("logFormat", accessLogFormat)
		b.Add("rollingSize", "10M")
		b.Add("keepDays", "30")
	}

	v.buildPHP(cfg)
	v.buildSSL(cfg)
	v.buildRealms(cfg)
	v.buildUpstreams(cfg)
	consumed := v.buildProxies(cfg)
	v.buildRootContext(cfg, consumed)
	v.buildAuthContexts(cfg, consumed)
	v.buildRedirects(cfg)

	if !v.Enable() {
		stop := cfg.AddBlock("context", stopURI)
		stop.Add("location", HTMLDir+"/")
		stop.Add("allowBrowse", "1")
	}

	rewriteFragments := v.buildIncludes(cfg)
	v.buildRewrite(cfg, rewriteFragments)

	return cfg
}

// buildPHP 只引用服务器级的脚本处理器行，外部应用与协议由 syncPHP 统一维护
func (v *baseVhost) buildPHP(cfg *Config) {
	if v.php == 0 {
		return
	}
	cfg.AddBlock("scripthandler", "").Add("include", phpHandlerFile(v.php))
}

func (v *baseVhost) buildSSL(cfg *Config) {
	if v.ssl == nil {
		return
	}
	quic := slices.ContainsFunc(v.listens, func(l types.Listen) bool {
		return slices.Contains(l.Args, "quic")
	})
	b := cfg.AddBlock("vhssl", "")
	b.Add("keyFile", v.ssl.Key)
	b.Add("certFile", v.ssl.Cert)
	b.Add("certChain", "1")
	b.Add("sslProtocol", strconv.Itoa(protocolBits(v.ssl.Protocols)))
	b.Add("sslSessionCache", "1")
	b.Add("enableQuic", map[bool]string{true: "1", false: "0"}[quic])
	if v.ssl.OCSP {
		b.Add("enableStapling", "1")
	}
}

// hstsHeader HSTS 响应头名
const hstsHeader = "Strict-Transport-Security"

// contextHeaders 需要写入每个上下文的响应头操作，OpenLiteSpeed 只在上下文级处理 extraHeaders
func (v *baseVhost) contextHeaders() []string {
	if v.ssl != nil && v.ssl.HSTS {
		return []string{"Header set " + hstsHeader + " max-age=31536000"}
	}
	return nil
}

// setHeaders 写入上下文的响应头操作
func setHeaders(ctx *Block, lines []string) {
	if len(lines) > 0 {
		ctx.Append(&Directive{Name: "extraHeaders", Value: strings.Join(lines, "\n"), Multiline: true})
	}
}

// buildRootContext 根上下文承载响应头、整站认证与 .htaccess 自动加载；反向代理已占用根路径时不再生成
func (v *baseVhost) buildRootContext(cfg *Config, consumed map[int]bool) {
	if cfg.Block("context", "/") != nil {
		return
	}
	ctx := cfg.AddBlock("context", "/")
	ctx.Add("location", "$DOC_ROOT/")
	ctx.Add("allowBrowse", "1")
	rw := ctx.AddBlock("rewrite", "")
	rw.Add("enable", "1")
	rw.Add("autoLoadHtaccess", "1")
	for i, auth := range v.auths {
		if auth.Path == "/" && !consumed[i] {
			ctx.Add("realm", v.realmName(i))
			ctx.AddMeta("auth", auth.Path)
			consumed[i] = true
			break
		}
	}
	setHeaders(ctx, v.contextHeaders())
}

func (v *baseVhost) realmName(i int) string {
	return fmt.Sprintf("%s_auth_%d", v.safeName, i)
}

func (v *baseVhost) buildRealms(cfg *Config) {
	for i, auth := range v.auths {
		db := cfg.AddBlock("realm", v.realmName(i)).AddBlock("userDB", "")
		db.Add("location", auth.UserFile)
		db.Add("maxCacheSize", "200")
		db.Add("cacheTimeout", "60")
	}
}

// buildAuthContexts 为未合并进代理上下文的认证路径生成静态上下文
func (v *baseVhost) buildAuthContexts(cfg *Config, consumed map[int]bool) {
	for i, auth := range v.auths {
		if consumed[i] {
			continue
		}
		uri := "/"
		if auth.Path != "/" {
			uri = "/" + strings.Trim(auth.Path, "/") + "/"
		}
		ctx := cfg.AddBlock("context", uri)
		ctx.Add("location", "$DOC_ROOT"+uri)
		ctx.Add("allowBrowse", "1")
		ctx.Add("realm", v.realmName(i))
		ctx.AddMeta("auth", auth.Path)
		setHeaders(ctx, v.contextHeaders())
	}
}

// buildIncludes 写入用户自定义 include 与片段 include，返回需放入 rewrite 块的重写规则片段
func (v *baseVhost) buildIncludes(cfg *Config) []string {
	for _, inc := range v.includes {
		cfg.Add("include", inc.Path)
	}

	var rewrites []string
	for _, scope := range []types.ConfigScope{types.ScopeShared, types.ScopeSite} {
		dir := filepath.Join(v.configDir, string(scope))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".conf") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			switch fragmentKind(string(content)) {
			case fragmentRewrite:
				rewrites = append(rewrites, path)
			case fragmentConfig:
				cfg.Add("include", path)
			}
		}
	}

	return rewrites
}

func (v *baseVhost) buildRewrite(cfg *Config, fragments []string) {
	rw := cfg.AddBlock("rewrite", "")
	rw.Add("enable", "1")

	// OLS 核心对 /.well-known/acme-challenge/ 强制跳过 rewrite，无需排除验证路径
	if !v.Enable() {
		rw.Add("RewriteCond", "%{REQUEST_URI} !^"+stopURI)
		rw.Add("RewriteRule", "^ "+stopURI+"stop.html [L]")
	}
	if v.ssl != nil && v.ssl.HTTPRedirect {
		rw.Add("RewriteCond", "%{HTTPS} !on")
		rw.Add("RewriteRule", "^(.*)$ https://%{HTTP_HOST}%{REQUEST_URI} [R=301,L]")
	}
	for _, r := range v.redirects {
		if r.Type != types.RedirectTypeHost {
			continue
		}
		to := r.To
		if r.KeepURI {
			to += "$1"
		}
		rw.Add("RewriteCond", fmt.Sprintf("%%{HTTP_HOST} ^%s$ [NC]", strings.ReplaceAll(r.From, ".", `\.`)))
		rw.Add("RewriteRule", fmt.Sprintf("^(.*)$ %s [R=%d,L]", to, redirectStatus(r)))
	}
	for _, path := range fragments {
		rw.Add("include", path)
	}
}

func (v *baseVhost) buildListen() *Config {
	cfg := &Config{}
	cfg.Append(&Comment{Text: " Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"})
	for _, l := range v.listens {
		cfg.Add("listen", strings.TrimSpace(l.Address+" "+strings.Join(l.Args, " ")))
	}
	cfg.Add("domain", strings.Join(v.domains, " "))
	return cfg
}

// ========== 辅助 ==========

type fragmentType int

const (
	fragmentEmpty   fragmentType = iota // 无有效指令
	fragmentRewrite                     // 全部为 Apache 风格重写规则，需放入 rewrite 块
	fragmentConfig                      // 普通 vhost 级配置
)

var rewriteDirectivePattern = regexp.MustCompile(`(?i)^rewrite(rule|cond|base|engine|options|map)\b`)

// fragmentKind 按内容判断片段类型
func fragmentKind(content string) fragmentType {
	kind := fragmentEmpty
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !rewriteDirectivePattern.MatchString(line) {
			return fragmentConfig
		}
		kind = fragmentRewrite
	}
	return kind
}

// safeName 将站点名转换为 OpenLiteSpeed 标识符安全形式
func safeName(name string) string {
	return strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			return char
		}
		return '_'
	}, name)
}

// parseIfExists 解析存在的配置文件，不存在返回 nil
func parseIfExists(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	return ParseFile(path)
}

// splitList 拆分逗号分隔的值
func splitList(value string) []string {
	out := make([]string, 0)
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// sslProtocol 位掩码：SSLv3=1 TLSv1.0=2 TLSv1.1=4 TLSv1.2=8 TLSv1.3=16
var protocolBitTable = []struct {
	name string
	bit  int
}{
	{"TLSv1", 2},
	{"TLSv1.1", 4},
	{"TLSv1.2", 8},
	{"TLSv1.3", 16},
}

func protocolBits(protocols []string) int {
	bits := 0
	for _, p := range protocols {
		for _, item := range protocolBitTable {
			if strings.EqualFold(p, item.name) || (item.name == "TLSv1" && strings.EqualFold(p, "TLSv1.0")) {
				bits |= item.bit
			}
		}
	}
	if bits == 0 {
		bits = 24
	}
	return bits
}

func protocolsFromBits(value string) []string {
	bits, _ := strconv.Atoi(value)
	if bits == 0 {
		bits = 24
	}
	var out []string
	for _, item := range protocolBitTable {
		if bits&item.bit != 0 {
			out = append(out, item.name)
		}
	}
	return out
}
