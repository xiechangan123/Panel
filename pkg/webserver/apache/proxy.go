package apache

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

var durationPattern = regexp.MustCompile(`^(\d+)([smhd]?)$`)

// parseDurationToSeconds 将时长字符串转换为秒数，支持 "10s" "5m" "1h" "1d"
func parseDurationToSeconds(duration string) int {
	duration = strings.TrimSpace(duration)
	if duration == "" {
		return 600 // 默认 10 分钟
	}

	matches := durationPattern.FindStringSubmatch(duration)
	if matches == nil {
		return 600
	}

	value, _ := strconv.Atoi(matches[1])
	switch matches[2] {
	case "m":
		return value * 60
	case "h":
		return value * 3600
	case "d":
		return value * 86400
	default:
		return value
	}
}

// proxyFilePattern 匹配代理配置文件名 (200-299)
var proxyFilePattern = regexp.MustCompile(`^(\d{3})-proxy\.conf$`)

// balancerFilePattern 匹配负载均衡配置文件名
var balancerFilePattern = regexp.MustCompile(`^(\d{3})-balancer-(.+)\.conf$`)

// parseProxyFiles 从 site 目录解析所有代理配置
func parseProxyFiles(siteDir string) ([]types.Proxy, error) {
	entries, err := os.ReadDir(siteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	proxies := make([]types.Proxy, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := proxyFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num < ProxyStartNum || num > ProxyEndNum {
			continue
		}

		proxy, err := parseProxyFile(filepath.Join(siteDir, entry.Name()))
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if proxy != nil {
			proxies = append(proxies, *proxy)
		}
	}

	return proxies, nil
}

// parseProxyFile 解析单个代理配置文件为结构体（基于 AST 遍历）
func parseProxyFile(filePath string) (*types.Proxy, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseFragment(string(content))
	if err != nil {
		return nil, err
	}

	proxy := &types.Proxy{
		Resolver: []string{},
		Headers:  make(map[string]string),
		Replaces: make(map[string]string),
	}

	// 匹配类型与目标路径在 Apache 语法里不可逆，回读取写入时留下的原值
	if blk := cfg.GetBlock("IfModule", "mod_proxy.c"); blk != nil {
		proxy.Location = blk.Meta("location")
		proxy.Pass = blk.Meta("pass")
	}
	if proxy.Location == "" {
		if d := cfg.FindOne("IfModule.ProxyPass"); d != nil && len(d.Args) >= 2 {
			proxy.Location = d.Arg(0)
			proxy.Pass = d.Arg(1)
		}
	}

	if d := cfg.FindOne("IfModule.LimitRequestBody"); d != nil {
		proxy.ClientMaxBodySize, _ = strconv.ParseInt(d.Arg(0), 10, 64)
	}
	if blk := cfg.FindOne("IfModule.Location"); blk != nil {
		proxy.AccessControl = parseAccessControl(blk)
		proxy.ResponseHeaders = parseResponseHeaders(blk)
		if d := blk.Get("LimitRequestBody"); d != nil {
			proxy.ClientMaxBodySize, _ = strconv.ParseInt(d.Arg(0), 10, 64)
		}
	}
	if d := cfg.FindOne("IfModule.SSLProxyVerify"); d != nil && strings.EqualFold(d.Arg(0), "require") {
		proxy.SSLBackend = &types.SSLBackendConfig{Verify: true}
		if ca := cfg.FindOne("IfModule.SSLProxyCACertificateFile"); ca != nil {
			proxy.SSLBackend.TrustedCertificate = ca.Arg(0)
		}
	}
	proxy.Timeout, proxy.Retry = parseProxyParams(cfg.FindOne("IfModule.ProxyPass"), cfg.FindOne("IfModule.ProxyPassMatch"))

	// RequestHeader set Host "host"（mod_proxy 直接子级）
	for _, d := range cfg.Find("IfModule.RequestHeader") {
		vals := d.Values()
		if len(vals) >= 3 && strings.EqualFold(vals[0], "set") && vals[1] == "Host" {
			proxy.Host = vals[2]
		}
	}
	if d := cfg.FindOne("IfModule.ProxyPreserveHost"); d != nil && strings.EqualFold(d.Arg(0), "on") {
		proxy.Host = "$host"
	}

	proxy.SNI = findSNIComment(cfg)

	if cfg.FindOne("IfModule.ProxyIOBufferSize") != nil {
		proxy.Buffering = true
	}

	// cache（mod_cache 子块，含 CacheEnable）
	for _, blk := range cfg.FindBlocks("IfModule.IfModule") {
		if blk.Has("CacheEnable") {
			proxy.Cache = parseCacheBlock(blk)
			break
		}
	}

	// 自定义请求头（mod_headers 子块，排除 Host）
	for _, d := range cfg.Find("IfModule.IfModule.RequestHeader") {
		vals := d.Values()
		if len(vals) >= 3 && strings.EqualFold(vals[0], "set") && vals[1] != "Host" {
			proxy.Headers[vals[1]] = parseHeaderValue(vals[2])
		}
	}

	for _, d := range cfg.Find("IfModule.IfModule.Substitute") {
		if len(d.Args) >= 1 {
			if from, to, ok := parseSubstitute(d.Args[0].Value); ok {
				proxy.Replaces[from] = to
			}
		}
	}

	return proxy, nil
}

// findSNIComment 从片段所有注释中提取 SNI 值
func findSNIComment(c *conf.Config) string {
	for _, cmt := range collectComments(c.Nodes) {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(cmt.Text), "SNI:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// collectComments 递归收集节点树中的所有注释
func collectComments(nodes []conf.Node) []*conf.Comment {
	var out []*conf.Comment
	for _, n := range nodes {
		switch v := n.(type) {
		case *conf.Comment:
			out = append(out, v)
		case *conf.Directive:
			if v.Block != nil {
				out = append(out, collectComments(v.Nodes)...)
			}
		}
	}
	return out
}

// parseCacheBlock 从 mod_cache 块提取缓存配置
func parseCacheBlock(blk *conf.Directive) *types.CacheConfig {
	cache := &types.CacheConfig{
		Valid:             make(map[string]string),
		NoCacheConditions: []string{},
		UseStale:          []string{},
		Methods:           []string{},
	}

	if d := blk.Get("CacheDefaultExpire"); d != nil && len(d.Args) > 0 {
		seconds, _ := strconv.Atoi(d.Args[0].Value)
		if minutes := seconds / 60; minutes > 0 {
			cache.Valid["any"] = fmt.Sprintf("%dm", minutes)
		} else {
			cache.Valid["any"] = fmt.Sprintf("%ds", seconds)
		}
	}

	return cache
}

// parseSubstitute 解析 Substitute 规则 s|from|to|flags，分隔符取规则第二个字符
func parseSubstitute(rule string) (from, to string, ok bool) {
	if len(rule) < 2 || rule[0] != 's' {
		return "", "", false
	}
	parts := strings.Split(rule[2:], string(rule[1]))
	if len(parts) >= 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// writeProxyFiles 将代理配置写入文件
func writeProxyFiles(siteDir string, proxies []types.Proxy, upstreams []string) error {
	if err := clearProxyFiles(siteDir); err != nil {
		return err
	}

	for i, proxy := range proxies {
		num := ProxyStartNum + i
		if num > ProxyEndNum {
			return fmt.Errorf("proxy rules exceed limit (%d)", ProxyEndNum-ProxyStartNum+1)
		}

		filePath := filepath.Join(siteDir, fmt.Sprintf("%03d-proxy.conf", num))
		if err := os.WriteFile(filePath, []byte(generateProxyConfig(proxy, upstreams)), 0600); err != nil {
			return fmt.Errorf("failed to write proxy config: %w", err)
		}
	}

	return nil
}

// clearProxyFiles 清除所有代理配置文件
func clearProxyFiles(siteDir string) error {
	entries, err := os.ReadDir(siteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := proxyFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num >= ProxyStartNum && num <= ProxyEndNum {
			if err := os.Remove(filepath.Join(siteDir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete proxy config: %w", err)
			}
		}
	}

	return nil
}

// location 描述 nginx 的 location 在 Apache 下的表达方式。
// Apache 的 <LocationMatch> 里写 ProxyPass 语法能过但运行时不代理，正则只能走 ProxyPassMatch
type location struct {
	regex   bool   // 用 ProxyPassMatch 而不是 ProxyPass
	pattern string // 前缀路径或正则
	path    string // 用于 ProxyPassReverse 与 <Location> 的路径前缀
}

// parseLocation 把 nginx 的匹配前缀翻译成 Apache 的匹配方式
func parseLocation(loc string) location {
	loc = strings.TrimSpace(loc)
	switch {
	case strings.HasPrefix(loc, "= "):
		path := normalizePath(strings.TrimPrefix(loc, "= "))
		return location{regex: true, pattern: "^" + regexp.QuoteMeta(path) + "$", path: path}
	case strings.HasPrefix(loc, "~* "):
		// PCRE 的内联标志，ProxyPassMatch 本身区分大小写
		return location{regex: true, pattern: "(?i)" + strings.TrimPrefix(loc, "~* "), path: "/"}
	case strings.HasPrefix(loc, "~ "):
		return location{regex: true, pattern: strings.TrimPrefix(loc, "~ "), path: "/"}
	default:
		path := normalizePath(strings.TrimPrefix(loc, "^~ "))
		return location{pattern: path, path: path}
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// proxyTarget 代理目标。nginx 的 proxy_pass 不带路径时保留完整 URI，带路径时替换掉 location 前缀，
// 而 Apache 的 ProxyPass 总是替换前缀，所以不带路径的要把 location 拼回去；
// ProxyPassMatch 会把整个 URI 附到目标后面，目标只能写到端口
func proxyTarget(pass string, loc location) string {
	origin, path := splitPass(pass)
	if loc.regex {
		return origin
	}
	target := pass
	if path == "" {
		target = origin + loc.path
	}
	// 两边的尾斜杠必须一致，否则深层路径会 500
	if strings.HasSuffix(loc.pattern, "/") {
		return strings.TrimSuffix(target, "/") + "/"
	}
	return strings.TrimSuffix(target, "/")
}

// nginx 变量到 Apache 表达式的映射，照搬会把 $remote_addr 这类变量当字面量发给上游
var headerVars = []struct{ from, to string }{
	{"$proxy_add_x_forwarded_for", "%{HTTP:X-Forwarded-For}"},
	{"$http_host", "%{HTTP_HOST}"},
	{"$remote_addr", "%{REMOTE_ADDR}"},
	{"$request_uri", "%{REQUEST_URI}"},
	{"$server_name", "%{SERVER_NAME}"},
	{"$server_port", "%{SERVER_PORT}"},
	{"$scheme", "%{REQUEST_SCHEME}"},
	{"$host", "%{HTTP_HOST}"},
}

// headerValue 含变量的头值转成 Apache 的 expr 写法
func headerValue(value string) string {
	replaced := value
	for _, v := range headerVars {
		replaced = strings.ReplaceAll(replaced, v.from, v.to)
	}
	if replaced == value {
		return value
	}
	return "expr=" + replaced
}

// parseHeaderValue 还原成面板的 nginx 变量写法
func parseHeaderValue(value string) string {
	expr, ok := strings.CutPrefix(value, "expr=")
	if !ok {
		return value
	}
	for _, v := range headerVars {
		expr = strings.ReplaceAll(expr, v.to, v.from)
	}
	return expr
}

// splitPass 拆出代理地址的源与路径部分
func splitPass(pass string) (string, string) {
	scheme, rest, found := strings.Cut(pass, "://")
	if !found {
		return pass, ""
	}
	host, path, found := strings.Cut(rest, "/")
	if !found {
		return pass, ""
	}
	return scheme + "://" + host, "/" + path
}

// balancerPass 上游名换成 Apache 的 balancer 地址，照搬 nginx 的 http://<上游名> 会被当真实主机解析
func balancerPass(pass string, upstreams []string) string {
	origin, path := splitPass(pass)
	_, host, found := strings.Cut(origin, "://")
	if !found || !slices.Contains(upstreams, host) {
		return pass
	}
	return "balancer://" + host + path
}

// proxyParams ProxyPass 的 worker 参数
func proxyParams(p types.Proxy) []string {
	var params []string
	if p.Timeout != nil {
		if p.Timeout.Connect > 0 {
			params = append(params, fmt.Sprintf("connectiontimeout=%d", int(p.Timeout.Connect.Seconds())))
		}
		if p.Timeout.Read > 0 {
			params = append(params, fmt.Sprintf("timeout=%d", int(p.Timeout.Read.Seconds())))
		}
	}
	if p.Retry != nil && p.Retry.Timeout > 0 {
		params = append(params, fmt.Sprintf("retry=%d", int(p.Retry.Timeout.Seconds())))
	}
	// 2.4.47 起 mod_proxy_http 自带协议升级，不加这个参数 WebSocket 会被当普通请求
	params = append(params, "upgrade=websocket")
	return params
}

// parseProxyParams 从 worker 参数还原超时与重试
func parseProxyParams(dirs ...*conf.Directive) (*types.TimeoutConfig, *types.RetryConfig) {
	var timeout *types.TimeoutConfig
	var retry *types.RetryConfig
	for _, d := range dirs {
		for _, a := range d.ArgsFrom(2) {
			key, value, found := strings.Cut(a, "=")
			seconds, err := strconv.Atoi(value)
			if !found || err != nil {
				continue
			}
			switch key {
			case "connectiontimeout":
				if timeout == nil {
					timeout = &types.TimeoutConfig{}
				}
				timeout.Connect = time.Duration(seconds) * time.Second
			case "timeout":
				if timeout == nil {
					timeout = &types.TimeoutConfig{}
				}
				timeout.Read = time.Duration(seconds) * time.Second
			case "retry":
				retry = &types.RetryConfig{Timeout: time.Duration(seconds) * time.Second}
			}
		}
	}
	return timeout, retry
}

// accessControlNodes IP 访问控制，deny 要包在 RequireAll 里才是"默认放行、排除名单"的语义
func accessControlNodes(ac *types.AccessControlConfig) []conf.Node {
	switch {
	case len(ac.Deny) == 0:
		return []conf.Node{conf.Dir("Require", append([]string{"ip"}, ac.Allow...)...)}
	default:
		all := conf.Blk("RequireAll")
		if len(ac.Allow) > 0 {
			all.Append(conf.Dir("Require", append([]string{"ip"}, ac.Allow...)...))
		} else {
			all.Append(conf.Dir("Require", "all", "granted"))
		}
		all.Append(conf.Dir("Require", append([]string{"not", "ip"}, ac.Deny...)...))
		return []conf.Node{all}
	}
}

func parseAccessControl(blk *conf.Directive) *types.AccessControlConfig {
	ac := &types.AccessControlConfig{Allow: []string{}, Deny: []string{}}
	collect := func(b *conf.Directive) {
		for _, d := range b.GetAll("Require") {
			switch {
			case d.Arg(0) == "not" && d.Arg(1) == "ip":
				ac.Deny = append(ac.Deny, d.ArgsFrom(2)...)
			case d.Arg(0) == "ip":
				ac.Allow = append(ac.Allow, d.ArgsFrom(1)...)
			}
		}
	}
	collect(blk)
	for _, all := range blk.Blocks("RequireAll") {
		collect(all)
	}
	if len(ac.Allow) == 0 && len(ac.Deny) == 0 {
		return nil
	}
	return ac
}

func parseResponseHeaders(blk *conf.Directive) *types.ResponseHeaderConfig {
	headers := &types.ResponseHeaderConfig{Hide: []string{}, Add: make(map[string]string)}
	for _, d := range blk.GetAll("Header") {
		args := d.Values()
		if len(args) > 0 && strings.EqualFold(args[0], "always") {
			args = args[1:]
		}
		switch {
		case len(args) >= 2 && strings.EqualFold(args[0], "unset"):
			headers.Hide = append(headers.Hide, args[1])
		case len(args) >= 3 && strings.EqualFold(args[0], "set"):
			headers.Add[args[1]] = parseHeaderValue(args[2])
		}
	}
	if len(headers.Hide) == 0 && len(headers.Add) == 0 {
		return nil
	}
	return headers
}

// generateProxyConfig 构建代理配置 AST 并序列化
func generateProxyConfig(proxy types.Proxy, upstreams []string) string {
	loc := parseLocation(proxy.Location)
	pass := balancerPass(proxy.Pass, upstreams)
	target := proxyTarget(pass, loc)

	inner := conf.Blk("IfModule", "mod_proxy.c")
	// 匹配类型与目标路径在 Apache 语法里不可逆，留下原值供回读
	inner.AddMeta("location", proxy.Location)
	inner.AddMeta("pass", proxy.Pass)

	args := []string{loc.pattern, target}
	// balancer 目标只认 balancer 参数，worker 参数得写在 BalancerMember 上
	if !strings.HasPrefix(target, "balancer://") {
		args = append(args, proxyParams(proxy)...)
	}
	if loc.regex {
		inner.Append(conf.Dir("ProxyPassMatch", args...))
	} else {
		inner.Append(conf.Dir("ProxyPass", args...))
	}
	inner.Append(conf.Dir("ProxyPassReverse", loc.path, target))

	switch host := strings.TrimSpace(proxy.Host); host {
	case "$host":
		inner.Append(conf.Dir("ProxyPreserveHost", "On"))
	case "", "$proxy_host":
	default:
		inner.Append(conf.Dir("RequestHeader", "set", "Host", host))
	}
	// mod_proxy 自带 X-Forwarded-For/Host/Server，这两个要自己补
	inner.Append(
		conf.Dir("RequestHeader", "set", "X-Real-IP", "expr=%{REMOTE_ADDR}"),
		conf.Dir("RequestHeader", "set", "X-Forwarded-Proto", "expr=%{REQUEST_SCHEME}"),
	)

	if strings.HasPrefix(target, "https://") {
		inner.Append(conf.Dir("SSLProxyEngine", "On"))
		if proxy.SSLBackend != nil && proxy.SSLBackend.Verify {
			inner.Append(conf.Dir("SSLProxyVerify", "require"))
			if proxy.SSLBackend.TrustedCertificate != "" {
				inner.Append(conf.Dir("SSLProxyCACertificateFile", proxy.SSLBackend.TrustedCertificate))
			}
		} else {
			inner.Append(
				conf.Dir("SSLProxyVerify", "none"),
				conf.Dir("SSLProxyCheckPeerCN", "off"),
				conf.Dir("SSLProxyCheckPeerName", "off"),
			)
		}
		if proxy.SNI != "" {
			// 垃圾 Apache 不支持自定义 SNI，写注释备注
			inner.Append(conf.Cmt("SNI: " + proxy.SNI))
		}
	}

	if proxy.Buffering {
		inner.Append(conf.Dir("ProxyIOBufferSize", "65536"))
	}

	if proxy.Cache != nil {
		expireSeconds := 600
		for _, duration := range proxy.Cache.Valid {
			expireSeconds = parseDurationToSeconds(duration)
			break
		}
		inner.Append(conf.Blk("IfModule", "mod_cache.c").Append(
			conf.Dir("CacheEnable", "disk", loc.path),
			conf.Dir("CacheDefaultExpire", strconv.Itoa(expireSeconds)),
		))
	}

	if len(proxy.Headers) > 0 {
		headers := conf.Blk("IfModule", "mod_headers.c")
		for _, name := range slices.Sorted(maps.Keys(proxy.Headers)) {
			headers.Append(conf.Dir("RequestHeader", "set", name, headerValue(proxy.Headers[name])))
		}
		inner.Append(headers)
	}

	if len(proxy.Replaces) > 0 {
		sub := conf.Blk("IfModule", "mod_substitute.c").Append(
			conf.Dir("AddOutputFilterByType", "SUBSTITUTE", "text/html", "text/plain", "text/xml"),
		)
		for _, from := range slices.Sorted(maps.Keys(proxy.Replaces)) {
			// 用 | 作为分隔符以支持含 / 的内容，强制双引号
			sub.Append(&conf.Directive{Name: "Substitute", Args: []conf.Arg{dquote(fmt.Sprintf("s|%s|%s|n", from, proxy.Replaces[from]))}})
		}
		inner.Append(sub)
	}

	// 这几项是按路径生效的，必须落在容器块里
	if scoped := scopedNodes(proxy); len(scoped) > 0 {
		inner.Append(conf.Blk("Location", loc.path).Append(scoped...))
	}

	cfg := &conf.Config{}
	cfg.Append(
		conf.Cmt("Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"),
		conf.Cmt(fmt.Sprintf("Reverse proxy: %s -> %s", proxy.Location, proxy.Pass)),
		inner,
	)
	return Export(cfg) + "\n"
}

// scopedNodes 访问控制、响应头与请求体限制
func scopedNodes(p types.Proxy) []conf.Node {
	var nodes []conf.Node
	if p.AccessControl != nil && (len(p.AccessControl.Allow) > 0 || len(p.AccessControl.Deny) > 0) {
		nodes = append(nodes, accessControlNodes(p.AccessControl)...)
	}
	if p.ResponseHeaders != nil {
		for _, name := range p.ResponseHeaders.Hide {
			nodes = append(nodes, conf.Dir("Header", "always", "unset", name))
		}
		for _, name := range slices.Sorted(maps.Keys(p.ResponseHeaders.Add)) {
			nodes = append(nodes, conf.Dir("Header", "always", "set", name, headerValue(p.ResponseHeaders.Add[name])))
		}
	}
	if p.ClientMaxBodySize > 0 {
		nodes = append(nodes, conf.Dir("LimitRequestBody", strconv.FormatInt(p.ClientMaxBodySize, 10)))
	}
	return nodes
}

// parseBalancerFiles 从 shared 目录解析所有负载均衡配置（Apache 的 upstream 等价物）
func parseBalancerFiles(sharedDir string) ([]types.Upstream, error) {
	entries, err := os.ReadDir(sharedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	upstreams := make([]types.Upstream, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := balancerFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		upstream, err := parseBalancerFile(filepath.Join(sharedDir, entry.Name()), matches[2])
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if upstream != nil {
			upstreams = append(upstreams, *upstream)
		}
	}

	return upstreams, nil
}

// parseBalancerFile 解析单个负载均衡配置文件为结构体（基于 AST 遍历）
func parseBalancerFile(filePath string, name string) (*types.Upstream, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseFragment(string(content))
	if err != nil {
		return nil, err
	}

	upstream := &types.Upstream{
		Name:     name,
		Servers:  make(map[string]string),
		Resolver: []string{},
	}

	for _, d := range cfg.Find("IfModule.Proxy.BalancerMember") {
		if len(d.Args) == 0 {
			continue
		}
		addr, options := parseMember(d.Values())
		upstream.Servers[addr] = options
	}

	// ProxySet lbmethod=xxx / max=N
	for _, d := range cfg.Find("IfModule.Proxy.ProxySet") {
		for _, a := range d.Args {
			k, v, found := strings.Cut(a.Value, "=")
			if !found {
				continue
			}
			switch k {
			case "lbmethod":
				upstream.Algo = algoFromLBMethod(v)
			case "max":
				upstream.Keepalive, _ = strconv.Atoi(v)
			}
		}
	}
	// 降级前的原值优先
	if blk := cfg.FindBlocks("IfModule.Proxy"); len(blk) > 0 {
		if algo := blk[0].Meta("algo"); algo != "" {
			upstream.Algo = types.NormalizeAlgo(algo)
		}
	}

	return upstream, nil
}

// lbMethods 规范算法名到 Apache lbmethod 的映射，Apache 只有按请求数、按流量、按繁忙度三种
var lbMethods = map[string]string{
	types.AlgoRoundRobin: "byrequests",
	types.AlgoLeastConn:  "bybusyness",
}

func algoFromLBMethod(method string) string {
	for algo, m := range lbMethods {
		if m == method {
			return algo
		}
	}
	return types.AlgoRoundRobin
}

// balancerMember 上游地址与参数转 Apache 语法：地址必须带协议前缀，
// 参数名与 nginx 完全不同，照搬过去会让 Apache 以 "unknown Worker parameter" 拒载
func balancerMember(addr, options string) []string {
	// 2.4.47 起 mod_proxy_http 自带协议升级，不加这个参数 WebSocket 会被当普通请求
	args := []string{balancerAddr(addr), "upgrade=websocket"}
	for _, opt := range strings.Fields(options) {
		key, value, _ := strings.Cut(opt, "=")
		switch key {
		case "weight":
			args = append(args, "loadfactor="+value)
		case "fail_timeout":
			args = append(args, "retry="+strings.TrimSuffix(value, "s"))
		case "backup":
			args = append(args, "status=+H")
		case "down":
			args = append(args, "status=+D")
		}
	}
	return args
}

func balancerAddr(addr string) string {
	switch {
	case strings.Contains(addr, "://"):
		return addr
	case strings.HasPrefix(addr, "unix:"):
		return addr + "|http://localhost"
	default:
		return "http://" + addr
	}
}

// parseMember 还原 BalancerMember 为面板的地址与 nginx 风格参数
func parseMember(args []string) (string, string) {
	addr := args[0]
	if unix, ok := strings.CutSuffix(addr, "|http://localhost"); ok {
		addr = unix
	} else {
		addr = strings.TrimPrefix(addr, "http://")
	}

	var options []string
	for _, a := range args[1:] {
		if a == "upgrade=websocket" {
			continue
		}
		key, value, _ := strings.Cut(a, "=")
		switch {
		case key == "loadfactor":
			options = append(options, "weight="+value)
		case key == "retry":
			options = append(options, "fail_timeout="+value+"s")
		case a == "status=+H":
			options = append(options, "backup")
		case a == "status=+D":
			options = append(options, "down")
		}
	}
	return addr, strings.Join(options, " ")
}

// writeBalancerFiles 将负载均衡配置写入文件
func writeBalancerFiles(sharedDir string, upstreams []types.Upstream) error {
	if err := clearBalancerFiles(sharedDir); err != nil {
		return err
	}

	for i, upstream := range upstreams {
		num := 100 + i
		filePath := filepath.Join(sharedDir, fmt.Sprintf("%03d-balancer-%s.conf", num, upstream.Name))
		if err := os.WriteFile(filePath, []byte(generateBalancerConfig(upstream)), 0600); err != nil {
			return fmt.Errorf("failed to write balancer config: %w", err)
		}
	}

	return nil
}

// clearBalancerFiles 清除所有负载均衡配置文件
func clearBalancerFiles(sharedDir string) error {
	entries, err := os.ReadDir(sharedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if balancerFilePattern.MatchString(entry.Name()) {
			if err := os.Remove(filepath.Join(sharedDir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete balancer config: %w", err)
			}
		}
	}

	return nil
}

// generateBalancerConfig 构建负载均衡配置 AST 并序列化
func generateBalancerConfig(upstream types.Upstream) string {
	proxy := conf.Blk("Proxy", "balancer://"+upstream.Name)
	for _, addr := range slices.Sorted(maps.Keys(upstream.Servers)) {
		proxy.Append(conf.Dir("BalancerMember", balancerMember(addr, upstream.Servers[addr])...))
	}

	algo := types.NormalizeAlgo(upstream.Algo)
	method, ok := lbMethods[algo]
	if !ok {
		method = lbMethods[types.AlgoRoundRobin]
		proxy.AddMeta("algo", algo)
	}
	proxy.Append(conf.Dir("ProxySet", "lbmethod="+method))
	if upstream.Keepalive > 0 {
		proxy.Append(conf.Dir("ProxySet", "max="+strconv.Itoa(upstream.Keepalive)))
	}

	cfg := &conf.Config{}
	cfg.Append(
		conf.Cmt("Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"),
		conf.Cmt("Load balancer: "+upstream.Name),
		conf.Blk("IfModule", "mod_proxy_balancer.c").Append(proxy),
	)
	return Export(cfg) + "\n"
}
