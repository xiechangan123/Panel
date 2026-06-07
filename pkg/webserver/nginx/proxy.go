package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/tufanbarisyildirim/gonginx/config"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// proxyFilePattern 匹配代理配置文件名 (200-299)
var proxyFilePattern = regexp.MustCompile(`^(\d{3})-proxy\.conf$`)

// firstParam 返回指令切片中首个指令的首个参数值
func firstParam(p *Parser, dirs []config.IDirective) string {
	if len(dirs) == 0 {
		return ""
	}
	params := p.parameters2Slices(dirs[0].GetParameters())
	if len(params) == 0 {
		return ""
	}
	return params[0]
}

// unquote 去除字符串两端的引号
func unquote(s string) string {
	return strings.Trim(s, `"'`)
}

// parseDuration 解析 Nginx 时间格式整串为 time.Duration，如 "5s" "5m" "5h"
func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	unit := ""
	if len(s) > 0 {
		if last := s[len(s)-1]; last < '0' || last > '9' {
			unit = string(last)
			s = s[:len(s)-1]
		}
	}
	value, _ := strconv.Atoi(s)
	switch unit {
	case "m":
		return time.Duration(value) * time.Minute
	case "h":
		return time.Duration(value) * time.Hour
	default:
		return time.Duration(value) * time.Second
	}
}

// parseSize 解析 Nginx 大小格式整串为字节数，如 "10m" "512k" "1g"
func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	unit := ""
	if len(s) > 0 {
		if last := s[len(s)-1]; last < '0' || last > '9' {
			unit = strings.ToLower(string(last))
			s = s[:len(s)-1]
		}
	}
	value, _ := strconv.ParseInt(s, 10, 64)
	switch unit {
	case "k":
		return value * 1024
	case "m":
		return value * 1024 * 1024
	case "g":
		return value * 1024 * 1024 * 1024
	default:
		return value
	}
}

// formatBytesToNginx 格式化字节数为 Nginx 大小格式
func formatBytesToNginx(bytes int64) string {
	if bytes == 0 {
		return "0"
	}
	if bytes%(1024*1024*1024) == 0 {
		return fmt.Sprintf("%dg", bytes/(1024*1024*1024))
	}
	if bytes%(1024*1024) == 0 {
		return fmt.Sprintf("%dm", bytes/(1024*1024))
	}
	if bytes%1024 == 0 {
		return fmt.Sprintf("%dk", bytes/1024)
	}
	return fmt.Sprintf("%d", bytes)
}

// formatDurationToNginx 格式化 time.Duration 为 Nginx 时间格式
func formatDurationToNginx(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	seconds := int(d.Seconds())
	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

// parseProxyFiles 从 site 目录解析所有代理配置
func parseProxyFiles(siteDir string) ([]types.Proxy, error) {
	entries, err := os.ReadDir(siteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var proxies []types.Proxy
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

		filePath := filepath.Join(siteDir, entry.Name())
		proxy, err := parseProxyFile(filePath)
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if proxy != nil {
			proxies = append(proxies, *proxy)
		}
	}

	return proxies, nil
}

// parseProxyFile 解析单个代理配置文件
func parseProxyFile(filePath string) (*types.Proxy, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	p, err := NewParserFromString(string(content))
	if err != nil {
		return nil, err
	}
	cfg := p.Config()

	// 解析 location 块
	// location / {
	//     proxy_pass http://backend;
	//     ...
	// }
	locations := cfg.FindDirectives("location")
	if len(locations) == 0 {
		return nil, nil
	}

	proxy := &types.Proxy{
		Location: strings.Join(p.parameters2Slices(locations[0].GetParameters()), " "),
		Resolver: []string{},
		Headers:  make(map[string]string),
		Replaces: make(map[string]string),
	}

	// 解析 proxy_pass
	if d := cfg.FindDirectives("proxy_pass"); len(d) > 0 {
		proxy.Pass = firstParam(p, d)
	}

	// 解析 proxy_set_header：Host 单独提取，其余非标准头收入 Headers
	standardHeaders := map[string]bool{
		"Host": true, "X-Real-IP": true, "X-Forwarded-For": true,
		"X-Forwarded-Proto": true, "Upgrade": true, "Connection": true,
		"Early-Data": true, "Accept-Encoding": true,
	}
	for _, d := range cfg.FindDirectives("proxy_set_header") {
		vals := p.parameters2Slices(d.GetParameters())
		if len(vals) < 2 {
			continue
		}
		name, value := vals[0], unquote(strings.Join(vals[1:], " "))
		if name == "Host" {
			proxy.Host = value
		} else if !standardHeaders[name] {
			proxy.Headers[name] = value
		}
	}

	// 解析 proxy_ssl_name (SNI)
	if d := cfg.FindDirectives("proxy_ssl_name"); len(d) > 0 {
		proxy.SNI = firstParam(p, d)
	}

	// 解析 proxy_buffering
	if d := cfg.FindDirectives("proxy_buffering"); len(d) > 0 {
		proxy.Buffering = firstParam(p, d) == "on"
	}

	// 解析 proxy_cache 及其子指令
	if d := cfg.FindDirectives("proxy_cache"); len(d) > 0 && firstParam(p, d) != "off" {
		proxy.Cache = parseProxyCache(p, cfg)
	}

	// 解析 resolver
	if d := cfg.FindDirectives("resolver"); len(d) > 0 {
		proxy.Resolver = p.parameters2Slices(d[0].GetParameters())
	}

	// 解析 resolver_timeout
	if d := cfg.FindDirectives("resolver_timeout"); len(d) > 0 {
		proxy.ResolverTimeout = parseDuration(firstParam(p, d))
	}

	// 解析 sub_filter (响应内容替换)
	for _, d := range cfg.FindDirectives("sub_filter") {
		vals := p.parameters2Slices(d.GetParameters())
		if len(vals) >= 2 {
			proxy.Replaces[unquote(vals[0])] = unquote(vals[1])
		}
	}

	// 解析 proxy_http_version
	if d := cfg.FindDirectives("proxy_http_version"); len(d) > 0 {
		proxy.HTTPVersion = firstParam(p, d)
	}

	// 解析超时配置
	var timeout types.TimeoutConfig
	hasTimeout := false
	if d := cfg.FindDirectives("proxy_connect_timeout"); len(d) > 0 {
		timeout.Connect = parseDuration(firstParam(p, d))
		hasTimeout = true
	}
	if d := cfg.FindDirectives("proxy_read_timeout"); len(d) > 0 {
		timeout.Read = parseDuration(firstParam(p, d))
		hasTimeout = true
	}
	if d := cfg.FindDirectives("proxy_send_timeout"); len(d) > 0 {
		timeout.Send = parseDuration(firstParam(p, d))
		hasTimeout = true
	}
	if hasTimeout {
		proxy.Timeout = &timeout
	}

	// 解析重试配置
	var retry types.RetryConfig
	hasRetry := false
	if d := cfg.FindDirectives("proxy_next_upstream"); len(d) > 0 {
		retry.Conditions = p.parameters2Slices(d[0].GetParameters())
		hasRetry = true
	}
	if d := cfg.FindDirectives("proxy_next_upstream_tries"); len(d) > 0 {
		retry.Tries, _ = strconv.Atoi(firstParam(p, d))
		hasRetry = true
	}
	if d := cfg.FindDirectives("proxy_next_upstream_timeout"); len(d) > 0 {
		retry.Timeout = parseDuration(firstParam(p, d))
		hasRetry = true
	}
	if hasRetry {
		proxy.Retry = &retry
	}

	// 解析 client_max_body_size
	if d := cfg.FindDirectives("client_max_body_size"); len(d) > 0 {
		proxy.ClientMaxBodySize = parseSize(firstParam(p, d))
	}

	// 解析 SSL 后端验证配置
	var sslBackend types.SSLBackendConfig
	hasSSLBackend := false
	if d := cfg.FindDirectives("proxy_ssl_verify"); len(d) > 0 {
		sslBackend.Verify = firstParam(p, d) == "on"
		hasSSLBackend = true
	}
	if d := cfg.FindDirectives("proxy_ssl_trusted_certificate"); len(d) > 0 {
		sslBackend.TrustedCertificate = firstParam(p, d)
		hasSSLBackend = true
	}
	if d := cfg.FindDirectives("proxy_ssl_verify_depth"); len(d) > 0 {
		sslBackend.VerifyDepth, _ = strconv.Atoi(firstParam(p, d))
		hasSSLBackend = true
	}
	if hasSSLBackend {
		proxy.SSLBackend = &sslBackend
	}

	// 解析响应头配置
	var responseHeaders types.ResponseHeaderConfig
	hasResponseHeaders := false
	if hide := cfg.FindDirectives("proxy_hide_header"); len(hide) > 0 {
		responseHeaders.Hide = lo.Map(hide, func(d config.IDirective, _ int) string {
			return firstParam(p, []config.IDirective{d})
		})
		hasResponseHeaders = true
	}
	if add := cfg.FindDirectives("add_header"); len(add) > 0 {
		responseHeaders.Add = make(map[string]string)
		for _, d := range add {
			vals := p.parameters2Slices(d.GetParameters())
			if len(vals) >= 2 {
				responseHeaders.Add[vals[0]] = unquote(vals[1])
			}
		}
		hasResponseHeaders = true
	}
	if hasResponseHeaders {
		proxy.ResponseHeaders = &responseHeaders
	}

	// 解析 IP 访问控制
	var accessControl types.AccessControlConfig
	hasAccessControl := false
	if allow := cfg.FindDirectives("allow"); len(allow) > 0 {
		accessControl.Allow = lo.Map(allow, func(d config.IDirective, _ int) string {
			return firstParam(p, []config.IDirective{d})
		})
		hasAccessControl = true
	}
	if deny := cfg.FindDirectives("deny"); len(deny) > 0 {
		accessControl.Deny = lo.Map(deny, func(d config.IDirective, _ int) string {
			return firstParam(p, []config.IDirective{d})
		})
		hasAccessControl = true
	}
	if hasAccessControl {
		proxy.AccessControl = &accessControl
	}

	return proxy, nil
}

// parseProxyCache 从配置中解析 proxy_cache 相关子指令
func parseProxyCache(p *Parser, cfg *config.Config) *types.CacheConfig {
	cache := &types.CacheConfig{
		Valid:             make(map[string]string),
		NoCacheConditions: []string{},
		UseStale:          []string{},
		Methods:           []string{},
	}

	// proxy_cache_valid：最后一个参数是时长，其余为状态码；仅时长则归入 any
	for _, d := range cfg.FindDirectives("proxy_cache_valid") {
		parts := p.parameters2Slices(d.GetParameters())
		if len(parts) >= 2 {
			cache.Valid[strings.Join(parts[:len(parts)-1], " ")] = parts[len(parts)-1]
		} else if len(parts) == 1 {
			cache.Valid["any"] = parts[0]
		}
	}

	// proxy_cache_bypass / proxy_no_cache（两者参数一致，取其一）
	if d := cfg.FindDirectives("proxy_cache_bypass"); len(d) > 0 {
		cache.NoCacheConditions = p.parameters2Slices(d[0].GetParameters())
	}

	// proxy_cache_use_stale
	if d := cfg.FindDirectives("proxy_cache_use_stale"); len(d) > 0 {
		cache.UseStale = p.parameters2Slices(d[0].GetParameters())
	}

	// proxy_cache_background_update
	if d := cfg.FindDirectives("proxy_cache_background_update"); len(d) > 0 {
		cache.BackgroundUpdate = firstParam(p, d) == "on"
	}

	// proxy_cache_lock
	if d := cfg.FindDirectives("proxy_cache_lock"); len(d) > 0 {
		cache.Lock = firstParam(p, d) == "on"
	}

	// proxy_cache_min_uses
	if d := cfg.FindDirectives("proxy_cache_min_uses"); len(d) > 0 {
		cache.MinUses, _ = strconv.Atoi(firstParam(p, d))
	}

	// proxy_cache_methods
	if d := cfg.FindDirectives("proxy_cache_methods"); len(d) > 0 {
		cache.Methods = p.parameters2Slices(d[0].GetParameters())
	}

	// proxy_cache_key（gonginx 保留引号，需去引号）
	if d := cfg.FindDirectives("proxy_cache_key"); len(d) > 0 {
		cache.Key = unquote(firstParam(p, d))
	}

	return cache
}

// writeProxyFiles 将代理配置写入文件
func writeProxyFiles(siteDir string, proxies []types.Proxy) error {
	// 删除现有的代理配置文件 (200-299)
	if err := clearProxyFiles(siteDir); err != nil {
		return err
	}

	// 写入新的配置文件
	for i, proxy := range proxies {
		num := ProxyStartNum + i
		if num > ProxyEndNum {
			return fmt.Errorf("proxy rules exceed limit (%d)", ProxyEndNum-ProxyStartNum+1)
		}

		fileName := fmt.Sprintf("%03d-proxy.conf", num)
		filePath := filepath.Join(siteDir, fileName)

		content := generateProxyConfig(proxy)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
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
			filePath := filepath.Join(siteDir, entry.Name())
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete proxy config: %w", err)
			}
		}
	}

	return nil
}

// generateProxyConfig 生成代理配置内容
func generateProxyConfig(proxy types.Proxy) string {
	var sb strings.Builder

	location := proxy.Location

	sb.WriteString("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n")
	_, _ = fmt.Fprintf(&sb, "# Reverse proxy: %s -> %s\n", location, proxy.Pass)
	_, _ = fmt.Fprintf(&sb, "location %s {\n", location)

	// IP 访问控制
	if proxy.AccessControl != nil {
		for _, ip := range proxy.AccessControl.Allow {
			_, _ = fmt.Fprintf(&sb, "    allow %s;\n", ip)
		}
		for _, ip := range proxy.AccessControl.Deny {
			_, _ = fmt.Fprintf(&sb, "    deny %s;\n", ip)
		}
	}

	// 请求体大小限制
	if proxy.ClientMaxBodySize > 0 {
		_, _ = fmt.Fprintf(&sb, "    client_max_body_size %s;\n", formatBytesToNginx(proxy.ClientMaxBodySize))
	}

	// resolver 配置
	if len(proxy.Resolver) > 0 {
		_, _ = fmt.Fprintf(&sb, "    resolver %s;\n", strings.Join(proxy.Resolver, " "))
		if proxy.ResolverTimeout > 0 {
			_, _ = fmt.Fprintf(&sb, "    resolver_timeout %ds;\n", int(proxy.ResolverTimeout.Seconds()))
		}
	}

	_, _ = fmt.Fprintf(&sb, "    proxy_pass %s;\n", proxy.Pass)

	// HTTP 协议版本
	httpVersion := lo.If(proxy.HTTPVersion != "", proxy.HTTPVersion).Else("1.1")
	_, _ = fmt.Fprintf(&sb, "    proxy_http_version %s;\n", httpVersion)

	// Host 头
	host := lo.If(proxy.Host == "" || proxy.Host == "$proxy_host", "$proxy_host").ElseF(func() string {
		return lo.If(strings.HasPrefix(proxy.Host, "$"), proxy.Host).Else("\"" + proxy.Host + "\"")
	})
	_, _ = fmt.Fprintf(&sb, "    proxy_set_header Host %s;\n", host)

	// 标准代理头
	sb.WriteString("    proxy_set_header X-Real-IP $remote_addr;\n")
	sb.WriteString("    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	sb.WriteString("    proxy_set_header X-Forwarded-Proto $scheme;\n")
	sb.WriteString("    proxy_set_header Upgrade $http_upgrade;\n")
	sb.WriteString("    proxy_set_header Connection $connection_upgrade;\n")
	sb.WriteString("    proxy_set_header Early-Data $ssl_early_data;\n")

	// SNI 配置
	if strings.HasPrefix(proxy.Pass, "https") {
		sb.WriteString("    proxy_ssl_protocols TLSv1.2 TLSv1.3;\n")
		sb.WriteString("    proxy_ssl_session_reuse off;\n")
		sb.WriteString("    proxy_ssl_server_name on;\n")
		_, _ = fmt.Fprintf(&sb, "    proxy_ssl_name %s;\n", lo.If(proxy.SNI != "", proxy.SNI).Else("$proxy_host"))

		// SSL 后端验证
		if proxy.SSLBackend != nil && proxy.SSLBackend.Verify {
			sb.WriteString("    proxy_ssl_verify on;\n")
			if proxy.SSLBackend.VerifyDepth > 0 {
				_, _ = fmt.Fprintf(&sb, "    proxy_ssl_verify_depth %d;\n", proxy.SSLBackend.VerifyDepth)
			}
			if proxy.SSLBackend.TrustedCertificate != "" {
				_, _ = fmt.Fprintf(&sb, "    proxy_ssl_trusted_certificate %s;\n", proxy.SSLBackend.TrustedCertificate)
			}
		}
	}

	// 超时配置
	if proxy.Timeout != nil {
		if proxy.Timeout.Connect > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_connect_timeout %s;\n", formatDurationToNginx(proxy.Timeout.Connect))
		}
		if proxy.Timeout.Read > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_read_timeout %s;\n", formatDurationToNginx(proxy.Timeout.Read))
		}
		if proxy.Timeout.Send > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_send_timeout %s;\n", formatDurationToNginx(proxy.Timeout.Send))
		}
	}

	// 重试配置
	if proxy.Retry != nil {
		if len(proxy.Retry.Conditions) > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_next_upstream %s;\n", strings.Join(proxy.Retry.Conditions, " "))
		}
		if proxy.Retry.Tries > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_next_upstream_tries %d;\n", proxy.Retry.Tries)
		}
		if proxy.Retry.Timeout > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_next_upstream_timeout %s;\n", formatDurationToNginx(proxy.Retry.Timeout))
		}
	}

	// Buffering 配置
	_, _ = fmt.Fprintf(&sb, "    proxy_buffering %s;\n", lo.If(proxy.Buffering, "on").Else("off"))

	// Cache 配置
	if proxy.Cache != nil {
		sb.WriteString("    proxy_cache cache_one;\n")

		// 缓存时长
		if len(proxy.Cache.Valid) > 0 {
			for codes, duration := range proxy.Cache.Valid {
				if codes == "any" {
					_, _ = fmt.Fprintf(&sb, "    proxy_cache_valid %s;\n", duration)
				} else {
					_, _ = fmt.Fprintf(&sb, "    proxy_cache_valid %s %s;\n", codes, duration)
				}
			}
		} else {
			// 默认缓存时长
			sb.WriteString("    proxy_cache_valid 200 302 10m;\n")
			sb.WriteString("    proxy_cache_valid 404 10s;\n")
		}

		// 不缓存条件
		if len(proxy.Cache.NoCacheConditions) > 0 {
			conditions := strings.Join(proxy.Cache.NoCacheConditions, " ")
			_, _ = fmt.Fprintf(&sb, "    proxy_cache_bypass %s;\n", conditions)
			_, _ = fmt.Fprintf(&sb, "    proxy_no_cache %s;\n", conditions)
		}

		// 过期缓存使用策略
		if len(proxy.Cache.UseStale) > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_cache_use_stale %s;\n", strings.Join(proxy.Cache.UseStale, " "))
		}

		// 后台更新
		if proxy.Cache.BackgroundUpdate {
			sb.WriteString("    proxy_cache_background_update on;\n")
		}

		// 缓存锁
		if proxy.Cache.Lock {
			sb.WriteString("    proxy_cache_lock on;\n")
		}

		// 最小请求次数
		if proxy.Cache.MinUses > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_cache_min_uses %d;\n", proxy.Cache.MinUses)
		}

		// 缓存方法
		if len(proxy.Cache.Methods) > 0 {
			_, _ = fmt.Fprintf(&sb, "    proxy_cache_methods %s;\n", strings.Join(proxy.Cache.Methods, " "))
		}

		// 自定义缓存键
		if proxy.Cache.Key != "" {
			_, _ = fmt.Fprintf(&sb, "    proxy_cache_key \"%s\";\n", proxy.Cache.Key)
		}
	}

	// 自定义请求头
	for name, value := range proxy.Headers {
		_, _ = fmt.Fprintf(&sb, "    proxy_set_header %s %s;\n", name, lo.If(strings.HasPrefix(value, "$"), value).Else("\""+value+"\""))
	}

	// 响应内容替换
	if len(proxy.Replaces) > 0 {
		sb.WriteString("    proxy_set_header Accept-Encoding \"\";\n")
		sb.WriteString("    sub_filter_once off;\n")
		for from, to := range proxy.Replaces {
			_, _ = fmt.Fprintf(&sb, "    sub_filter \"%s\" \"%s\";\n", from, to)
		}
	}

	// 响应头修改
	if proxy.ResponseHeaders != nil {
		// 隐藏响应头
		for _, header := range proxy.ResponseHeaders.Hide {
			_, _ = fmt.Fprintf(&sb, "    proxy_hide_header %s;\n", header)
		}
		// 添加响应头
		for name, value := range proxy.ResponseHeaders.Add {
			formattedValue := lo.If(strings.HasPrefix(value, "$"), value).Else("\"" + value + "\"")
			_, _ = fmt.Fprintf(&sb, "    add_header %s %s always;\n", name, formattedValue)
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}
