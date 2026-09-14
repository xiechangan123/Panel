package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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

	if d := cfg.FindOne("IfModule.ProxyPass"); d != nil && len(d.Args) >= 2 {
		proxy.Location = d.Args[0].Value
		proxy.Pass = d.Args[1].Value
	}

	// RequestHeader set Host "host"（mod_proxy 直接子级）
	for _, d := range cfg.Find("IfModule.RequestHeader") {
		vals := d.Values()
		if len(vals) >= 3 && strings.EqualFold(vals[0], "set") && vals[1] == "Host" {
			proxy.Host = vals[2]
		}
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
			proxy.Headers[vals[1]] = vals[2]
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
func writeProxyFiles(siteDir string, proxies []types.Proxy) error {
	if err := clearProxyFiles(siteDir); err != nil {
		return err
	}

	for i, proxy := range proxies {
		num := ProxyStartNum + i
		if num > ProxyEndNum {
			return fmt.Errorf("proxy rules exceed limit (%d)", ProxyEndNum-ProxyStartNum+1)
		}

		filePath := filepath.Join(siteDir, fmt.Sprintf("%03d-proxy.conf", num))
		if err := os.WriteFile(filePath, []byte(generateProxyConfig(proxy)), 0600); err != nil {
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

// normalizeLocation 将 nginx 风格的 location 规整为 Apache 路径
func normalizeLocation(location string) string {
	for _, prefix := range []string{"^~ ", "~ ", "= "} {
		location = strings.TrimPrefix(location, prefix)
	}
	location = strings.TrimPrefix(location, "^")
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}
	return location
}

// generateProxyConfig 构建代理配置 AST 并序列化
func generateProxyConfig(proxy types.Proxy) string {
	location := normalizeLocation(proxy.Location)
	pass := proxy.Pass
	// 垃圾 Apache 要求 Pass 以 / 结尾才不报错
	if !strings.HasSuffix(pass, "/") {
		pass += "/"
	}

	inner := conf.Blk("IfModule", "mod_proxy.c").Append(
		conf.Dir("ProxyPass", location, pass),
		conf.Dir("ProxyPassReverse", location, pass),
	)

	if proxy.Host != "" {
		inner.Append(conf.Dir("RequestHeader", "set", "Host", proxy.Host))
	} else {
		inner.Append(conf.Dir("ProxyPreserveHost", "On"))
	}

	if proxy.SNI != "" || strings.HasPrefix(pass, "https://") {
		inner.Append(
			conf.Dir("SSLProxyEngine", "On"),
			conf.Dir("SSLProxyVerify", "none"),
			conf.Dir("SSLProxyCheckPeerCN", "off"),
			conf.Dir("SSLProxyCheckPeerName", "off"),
		)
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
			conf.Dir("CacheEnable", "disk", location),
			conf.Dir("CacheDefaultExpire", strconv.Itoa(expireSeconds)),
		))
	}

	if len(proxy.Headers) > 0 {
		headers := conf.Blk("IfModule", "mod_headers.c")
		for name, value := range proxy.Headers {
			headers.Append(conf.Dir("RequestHeader", "set", name, value))
		}
		inner.Append(headers)
	}

	if len(proxy.Replaces) > 0 {
		sub := conf.Blk("IfModule", "mod_substitute.c").Append(
			conf.Dir("AddOutputFilterByType", "SUBSTITUTE", "text/html", "text/plain", "text/xml"),
		)
		for from, to := range proxy.Replaces {
			// 用 | 作为分隔符以支持含 / 的内容，强制双引号
			sub.Append(&conf.Directive{Name: "Substitute", Args: []conf.Arg{dquote(fmt.Sprintf("s|%s|%s|n", from, to))}})
		}
		inner.Append(sub)
	}

	cfg := &conf.Config{}
	cfg.Append(
		conf.Cmt("Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"),
		conf.Cmt(fmt.Sprintf("Reverse proxy: %s -> %s", location, pass)),
		inner,
	)
	return Export(cfg) + "\n"
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
		vals := d.Values()
		if len(vals) == 0 {
			continue
		}
		upstream.Servers[vals[0]] = strings.Join(vals[1:], " ")
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
				if v != "byrequests" { // byrequests 为默认值，不存储
					upstream.Algo = v
				}
			case "max":
				upstream.Keepalive, _ = strconv.Atoi(v)
			}
		}
	}

	return upstream, nil
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
	for addr, options := range upstream.Servers {
		args := []string{addr}
		if options != "" {
			args = append(args, strings.Fields(options)...)
		}
		proxy.Append(conf.Dir("BalancerMember", args...))
	}

	algo := upstream.Algo
	if algo == "" {
		algo = "byrequests"
	}
	proxy.Append(conf.Dir("ProxySet", "lbmethod="+algo))
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
