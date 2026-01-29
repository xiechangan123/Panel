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

	"github.com/acepanel/panel/pkg/webserver/types"
)

// proxyFilePattern 匹配代理配置文件名 (200-299)
var proxyFilePattern = regexp.MustCompile(`^(\d{3})-proxy\.conf$`)

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

	contentStr := string(content)

	// 解析 location 块
	// location / {
	//     proxy_pass http://backend;
	//     ...
	// }
	locationPattern := regexp.MustCompile(`location\s+([^{]+)\{([^}]+(?:\{[^}]*}[^}]*)*)}`)
	matches := locationPattern.FindStringSubmatch(contentStr)
	if matches == nil {
		return nil, nil
	}

	proxy := &types.Proxy{
		Location: strings.TrimSpace(matches[1]),
		Resolver: []string{},
		Headers:  make(map[string]string),
		Replaces: make(map[string]string),
	}

	blockContent := matches[2]

	// 解析 proxy_pass
	passPattern := regexp.MustCompile(`proxy_pass\s+([^;]+);`)
	if pm := passPattern.FindStringSubmatch(blockContent); pm != nil {
		proxy.Pass = strings.TrimSpace(pm[1])
	}

	// 解析 proxy_set_header Host
	hostPattern := regexp.MustCompile(`proxy_set_header\s+Host\s+([^;]+);`)
	if hm := hostPattern.FindStringSubmatch(blockContent); hm != nil {
		host := strings.TrimSpace(hm[1])
		// 移除引号
		host = strings.Trim(host, `"'`)
		proxy.Host = host
	}

	// 解析 proxy_ssl_name (SNI)
	sniPattern := regexp.MustCompile(`proxy_ssl_name\s+([^;]+);`)
	if sm := sniPattern.FindStringSubmatch(blockContent); sm != nil {
		proxy.SNI = strings.TrimSpace(sm[1])
	}

	// 解析 proxy_buffering
	bufferingPattern := regexp.MustCompile(`proxy_buffering\s+(on|off);`)
	if bm := bufferingPattern.FindStringSubmatch(blockContent); bm != nil {
		proxy.Buffering = bm[1] == "on"
	}

	// 解析 proxy_cache
	cachePattern := regexp.MustCompile(`proxy_cache\s+(\S+);`)
	if cm := cachePattern.FindStringSubmatch(blockContent); cm != nil && cm[1] != "off" {
		proxy.Cache = &types.CacheConfig{
			Valid:             make(map[string]string),
			NoCacheConditions: []string{},
			UseStale:          []string{},
			Methods:           []string{},
		}

		// 解析 proxy_cache_valid
		cacheValidPattern := regexp.MustCompile(`proxy_cache_valid\s+([^;]+);`)
		cacheValidMatches := cacheValidPattern.FindAllStringSubmatch(blockContent, -1)
		for _, cvm := range cacheValidMatches {
			parts := strings.Fields(cvm[1])
			if len(parts) >= 2 {
				// 最后一个是时长，前面的是状态码
				duration := parts[len(parts)-1]
				codes := strings.Join(parts[:len(parts)-1], " ")
				proxy.Cache.Valid[codes] = duration
			} else if len(parts) == 1 {
				// 只有时长，表示 any
				proxy.Cache.Valid["any"] = parts[0]
			}
		}

		// 解析 proxy_cache_bypass / proxy_no_cache
		bypassPattern := regexp.MustCompile(`proxy_cache_bypass\s+([^;]+);`)
		if bm := bypassPattern.FindStringSubmatch(blockContent); bm != nil {
			proxy.Cache.NoCacheConditions = strings.Fields(bm[1])
		}

		// 解析 proxy_cache_use_stale
		useStalePattern := regexp.MustCompile(`proxy_cache_use_stale\s+([^;]+);`)
		if usm := useStalePattern.FindStringSubmatch(blockContent); usm != nil {
			proxy.Cache.UseStale = strings.Fields(usm[1])
		}

		// 解析 proxy_cache_background_update
		bgUpdatePattern := regexp.MustCompile(`proxy_cache_background_update\s+(on|off);`)
		if bgm := bgUpdatePattern.FindStringSubmatch(blockContent); bgm != nil {
			proxy.Cache.BackgroundUpdate = bgm[1] == "on"
		}

		// 解析 proxy_cache_lock
		lockPattern := regexp.MustCompile(`proxy_cache_lock\s+(on|off);`)
		if lm := lockPattern.FindStringSubmatch(blockContent); lm != nil {
			proxy.Cache.Lock = lm[1] == "on"
		}

		// 解析 proxy_cache_min_uses
		minUsesPattern := regexp.MustCompile(`proxy_cache_min_uses\s+(\d+);`)
		if mum := minUsesPattern.FindStringSubmatch(blockContent); mum != nil {
			proxy.Cache.MinUses, _ = strconv.Atoi(mum[1])
		}

		// 解析 proxy_cache_methods
		methodsPattern := regexp.MustCompile(`proxy_cache_methods\s+([^;]+);`)
		if mm := methodsPattern.FindStringSubmatch(blockContent); mm != nil {
			proxy.Cache.Methods = strings.Fields(mm[1])
		}

		// 解析 proxy_cache_key
		keyPattern := regexp.MustCompile(`proxy_cache_key\s+"?([^";]+)"?;`)
		if km := keyPattern.FindStringSubmatch(blockContent); km != nil {
			proxy.Cache.Key = strings.TrimSpace(km[1])
		}
	}

	// 解析 resolver
	resolverPattern := regexp.MustCompile(`resolver\s+([^;]+);`)
	if rm := resolverPattern.FindStringSubmatch(blockContent); rm != nil {
		parts := strings.Fields(rm[1])
		proxy.Resolver = parts
	}

	// 解析 resolver_timeout
	resolverTimeoutPattern := regexp.MustCompile(`resolver_timeout\s+(\d+)([smh]?);`)
	if rtm := resolverTimeoutPattern.FindStringSubmatch(blockContent); rtm != nil {
		value, _ := strconv.Atoi(rtm[1])
		unit := rtm[2]
		switch unit {
		case "m":
			proxy.ResolverTimeout = time.Duration(value) * time.Minute
		case "h":
			proxy.ResolverTimeout = time.Duration(value) * time.Hour
		default:
			proxy.ResolverTimeout = time.Duration(value) * time.Second
		}
	}

	// 解析 sub_filter (响应内容替换)
	subFilterPattern := regexp.MustCompile(`sub_filter\s+"([^"]+)"\s+"([^"]*)";`)
	subFilterMatches := subFilterPattern.FindAllStringSubmatch(blockContent, -1)
	for _, sfm := range subFilterMatches {
		proxy.Replaces[sfm[1]] = sfm[2]
	}

	// 解析自定义请求头
	standardHeaders := map[string]bool{
		"Host": true, "X-Real-IP": true, "X-Forwarded-For": true,
		"X-Forwarded-Proto": true, "Upgrade": true, "Connection": true,
		"Early-Data": true, "Accept-Encoding": true,
	}
	headerPattern := regexp.MustCompile(`proxy_set_header\s+(\S+)\s+"?([^";]+)"?;`)
	headerMatches := headerPattern.FindAllStringSubmatch(blockContent, -1)
	for _, hm := range headerMatches {
		headerName := strings.TrimSpace(hm[1])
		headerValue := strings.TrimSpace(hm[2])
		// 排除标准头
		if !standardHeaders[headerName] {
			proxy.Headers[headerName] = headerValue
		}
	}

	return proxy, nil
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
	sb.WriteString(fmt.Sprintf("# Reverse proxy: %s -> %s\n", location, proxy.Pass))
	sb.WriteString(fmt.Sprintf("location %s {\n", location))

	// resolver 配置
	if len(proxy.Resolver) > 0 {
		sb.WriteString(fmt.Sprintf("    resolver %s;\n", strings.Join(proxy.Resolver, " ")))
		if proxy.ResolverTimeout > 0 {
			sb.WriteString(fmt.Sprintf("    resolver_timeout %ds;\n", int(proxy.ResolverTimeout.Seconds())))
		}
	}

	sb.WriteString(fmt.Sprintf("    proxy_pass %s;\n", proxy.Pass))
	sb.WriteString("    proxy_http_version 1.1;\n")

	// Host 头
	host := lo.If(proxy.Host == "" || proxy.Host == "$proxy_host", "$proxy_host").ElseF(func() string {
		return lo.If(strings.HasPrefix(proxy.Host, "$"), proxy.Host).Else("\"" + proxy.Host + "\"")
	})
	sb.WriteString(fmt.Sprintf("    proxy_set_header Host %s;\n", host))

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
		sb.WriteString(fmt.Sprintf("    proxy_ssl_name %s;\n", lo.If(proxy.SNI != "", proxy.SNI).Else("$proxy_host")))
	}

	// Buffering 配置
	sb.WriteString(fmt.Sprintf("    proxy_buffering %s;\n", lo.If(proxy.Buffering, "on").Else("off")))

	// Cache 配置
	if proxy.Cache != nil {
		sb.WriteString("    proxy_cache cache_one;\n")

		// 缓存时长
		if len(proxy.Cache.Valid) > 0 {
			for codes, duration := range proxy.Cache.Valid {
				if codes == "any" {
					sb.WriteString(fmt.Sprintf("    proxy_cache_valid %s;\n", duration))
				} else {
					sb.WriteString(fmt.Sprintf("    proxy_cache_valid %s %s;\n", codes, duration))
				}
			}
		} else {
			// 默认缓存时长
			sb.WriteString("    proxy_cache_valid 200 302 10m;\n")
			sb.WriteString("    proxy_cache_valid 404 1m;\n")
		}

		// 不缓存条件
		if len(proxy.Cache.NoCacheConditions) > 0 {
			conditions := strings.Join(proxy.Cache.NoCacheConditions, " ")
			sb.WriteString(fmt.Sprintf("    proxy_cache_bypass %s;\n", conditions))
			sb.WriteString(fmt.Sprintf("    proxy_no_cache %s;\n", conditions))
		}

		// 过期缓存使用策略
		if len(proxy.Cache.UseStale) > 0 {
			sb.WriteString(fmt.Sprintf("    proxy_cache_use_stale %s;\n", strings.Join(proxy.Cache.UseStale, " ")))
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
			sb.WriteString(fmt.Sprintf("    proxy_cache_min_uses %d;\n", proxy.Cache.MinUses))
		}

		// 缓存方法
		if len(proxy.Cache.Methods) > 0 {
			sb.WriteString(fmt.Sprintf("    proxy_cache_methods %s;\n", strings.Join(proxy.Cache.Methods, " ")))
		}

		// 自定义缓存键
		if proxy.Cache.Key != "" {
			sb.WriteString(fmt.Sprintf("    proxy_cache_key \"%s\";\n", proxy.Cache.Key))
		}
	}

	// 自定义请求头
	for name, value := range proxy.Headers {
		sb.WriteString(fmt.Sprintf("    proxy_set_header %s %s;\n", name, lo.If(strings.HasPrefix(value, "$"), value).Else("\""+value+"\"")))
	}

	// 响应内容替换
	if len(proxy.Replaces) > 0 {
		sb.WriteString("    proxy_set_header Accept-Encoding \"\";\n")
		sb.WriteString("    sub_filter_once off;\n")
		for from, to := range proxy.Replaces {
			sb.WriteString(fmt.Sprintf("    sub_filter \"%s\" \"%s\";\n", from, to))
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}
