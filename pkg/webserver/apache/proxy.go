package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/acepanel/panel/pkg/webserver/types"
	"github.com/samber/lo"
)

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
	proxy := &types.Proxy{
		Resolver: []string{},
		Headers:  make(map[string]string),
		Replaces: make(map[string]string),
	}

	// 解析 ProxyPass 指令
	// ProxyPass / http://backend/
	proxyPassPattern := regexp.MustCompile(`ProxyPass\s+(\S+)\s+(\S+)`)
	if matches := proxyPassPattern.FindStringSubmatch(contentStr); matches != nil {
		proxy.Location = matches[1]
		proxy.Pass = matches[2]
	}

	// 解析 ProxyPreserveHost
	_ = regexp.MustCompile(`ProxyPreserveHost\s+On`).MatchString(contentStr)

	// 解析 RequestHeader set Host
	hostPattern := regexp.MustCompile(`RequestHeader\s+set\s+Host\s+"([^"]+)"`)
	if matches := hostPattern.FindStringSubmatch(contentStr); matches != nil {
		proxy.Host = matches[1]
	}

	// 解析 SSLProxyEngine 和 SNI
	if regexp.MustCompile(`SSLProxyEngine\s+On`).MatchString(contentStr) {
		// 尝试从 # SNI: xxx 注释中获取 SNI
		sniPattern := regexp.MustCompile(`#\s*SNI:\s*(\S+)`)
		if sm := sniPattern.FindStringSubmatch(contentStr); sm != nil {
			proxy.SNI = sm[1]
		}
	}

	// 解析 ProxyIOBufferSize (buffering)
	if regexp.MustCompile(`ProxyIOBufferSize`).MatchString(contentStr) {
		proxy.Buffering = true
	}

	// 解析 CacheEnable
	if regexp.MustCompile(`CacheEnable`).MatchString(contentStr) {
		proxy.Cache = true
	}

	// 解析 Substitute (响应内容替换)
	// 格式: Substitute "s|from|to|n" 或 Substitute "s/from/to/n"
	// 使用 | 作为分隔符以支持包含 / 的内容
	subPatternPipe := regexp.MustCompile(`Substitute\s+"s\|([^|]*)\|([^|]*)\|[gin]*"`)
	subMatchesPipe := subPatternPipe.FindAllStringSubmatch(contentStr, -1)
	for _, sm := range subMatchesPipe {
		proxy.Replaces[sm[1]] = sm[2]
	}

	// 解析自定义请求头 (排除 Host)
	headerPattern := regexp.MustCompile(`RequestHeader\s+set\s+(\S+)\s+"([^"]+)"`)
	headerMatches := headerPattern.FindAllStringSubmatch(contentStr, -1)
	for _, hm := range headerMatches {
		headerName := hm[1]
		headerValue := hm[2]
		if headerName != "Host" {
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
	// 将 Nginx 风格的 location 转换为 Apache 格式
	// ^~ / -> /
	// ~ ^/api -> /api (正则匹配需要使用 ProxyPassMatch)
	location = strings.TrimPrefix(location, "^~ ")
	location = strings.TrimPrefix(location, "~ ")
	location = strings.TrimPrefix(location, "= ")
	// 如果 location 以 ^ 开头（正则），去掉它
	location = strings.TrimPrefix(location, "^")
	// 确保 location 以 / 开头
	if !strings.HasPrefix(location, "/") {
		location = "/" + location
	}

	// Pass 不以 / 结尾时，添加 /
	// 垃圾 Apache 需要这样才不报错
	if !strings.HasSuffix(proxy.Pass, "/") {
		proxy.Pass += "/"
	}

	sb.WriteString("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n")
	sb.WriteString(fmt.Sprintf("# Reverse proxy: %s -> %s\n", location, proxy.Pass))

	// 启用代理模块
	sb.WriteString("<IfModule mod_proxy.c>\n")

	// ProxyPass 和 ProxyPassReverse
	sb.WriteString(fmt.Sprintf("    ProxyPass %s %s\n", location, proxy.Pass))
	sb.WriteString(fmt.Sprintf("    ProxyPassReverse %s %s\n", location, proxy.Pass))

	// Host 配置
	sb.WriteString(lo.If(proxy.Host != "", fmt.Sprintf("    RequestHeader set Host \"%s\"\n", proxy.Host)).Else("    ProxyPreserveHost On\n"))

	// SSL/SNI 配置
	if proxy.SNI != "" || strings.HasPrefix(proxy.Pass, "https://") {
		sb.WriteString("    SSLProxyEngine On\n")
		sb.WriteString("    SSLProxyVerify none\n")
		sb.WriteString("    SSLProxyCheckPeerCN off\n")
		sb.WriteString("    SSLProxyCheckPeerName off\n")
		if proxy.SNI != "" {
			// 垃圾 Apache 不支持自定义 SNI，写这里只是备注一下
			sb.WriteString(fmt.Sprintf("    # SNI: %s\n", proxy.SNI))
		}
	}

	// Buffering 配置
	if proxy.Buffering {
		sb.WriteString("    ProxyIOBufferSize 65536\n")
	}

	// Cache 配置
	if proxy.Cache {
		sb.WriteString("    <IfModule mod_cache.c>\n")
		sb.WriteString(fmt.Sprintf("        CacheEnable disk %s\n", location))
		sb.WriteString("        CacheDefaultExpire 600\n")
		sb.WriteString("    </IfModule>\n")
	}

	// 自定义请求头
	if len(proxy.Headers) > 0 {
		sb.WriteString("    <IfModule mod_headers.c>\n")
		for name, value := range proxy.Headers {
			sb.WriteString(fmt.Sprintf("        RequestHeader set %s \"%s\"\n", name, value))
		}
		sb.WriteString("    </IfModule>\n")
	}

	// 响应内容替换
	if len(proxy.Replaces) > 0 {
		sb.WriteString("    <IfModule mod_substitute.c>\n")
		sb.WriteString("        AddOutputFilterByType SUBSTITUTE text/html text/plain text/xml\n")
		for from, to := range proxy.Replaces {
			// 使用 | 作为分隔符以支持包含 / 的内容
			sb.WriteString(fmt.Sprintf("        Substitute \"s|%s|%s|n\"\n", from, to))
		}
		sb.WriteString("    </IfModule>\n")
	}

	sb.WriteString("</IfModule>\n")

	return sb.String()
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

		filePath := filepath.Join(sharedDir, entry.Name())
		upstream, err := parseBalancerFile(filePath, matches[2])
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if upstream != nil {
			upstreams = append(upstreams, *upstream)
		}
	}

	return upstreams, nil
}

// parseBalancerFile 解析单个负载均衡配置文件
func parseBalancerFile(filePath string, name string) (*types.Upstream, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)
	upstream := &types.Upstream{
		Name:     name,
		Servers:  make(map[string]string),
		Resolver: []string{},
	}

	// 解析 <Proxy balancer://name> 块
	// <Proxy balancer://backend>
	//     BalancerMember http://127.0.0.1:8080 loadfactor=5
	//     BalancerMember http://127.0.0.1:8081 loadfactor=3
	//     ProxySet lbmethod=byrequests
	// </Proxy>

	// 解析 BalancerMember
	memberPattern := regexp.MustCompile(`^\s*BalancerMember\s+(\S+)(?:\s+(.*))?$`)
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		if mm := memberPattern.FindStringSubmatch(line); mm != nil {
			addr := mm[1]
			options := ""
			if len(mm) > 2 {
				options = strings.TrimSpace(mm[2])
			}
			upstream.Servers[addr] = options
		}
	}

	// 解析负载均衡方法
	lbMethodPattern := regexp.MustCompile(`lbmethod=(\S+)`)
	if lm := lbMethodPattern.FindStringSubmatch(contentStr); lm != nil {
		// byrequests 是默认值，不需要存储
		if lm[1] != "byrequests" {
			upstream.Algo = lm[1]
		}
	}

	// 解析连接池大小 (类似 keepalive)
	maxPattern := regexp.MustCompile(`max=(\d+)`)
	if mm := maxPattern.FindStringSubmatch(contentStr); mm != nil {
		upstream.Keepalive, _ = strconv.Atoi(mm[1])
	}

	return upstream, nil
}

// writeBalancerFiles 将负载均衡配置写入文件
func writeBalancerFiles(sharedDir string, upstreams []types.Upstream) error {
	// 删除现有的负载均衡配置文件
	if err := clearBalancerFiles(sharedDir); err != nil {
		return err
	}

	// 写入新的配置文件，保持顺序
	for i, upstream := range upstreams {
		num := 100 + i
		fileName := fmt.Sprintf("%03d-balancer-%s.conf", num, upstream.Name)
		filePath := filepath.Join(sharedDir, fileName)

		content := generateBalancerConfig(upstream)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
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
			filePath := filepath.Join(sharedDir, entry.Name())
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete balancer config: %w", err)
			}
		}
	}

	return nil
}

// generateBalancerConfig 生成负载均衡配置内容
func generateBalancerConfig(upstream types.Upstream) string {
	var sb strings.Builder

	sb.WriteString("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n")
	sb.WriteString(fmt.Sprintf("# Load balancer: %s\n", upstream.Name))
	sb.WriteString("<IfModule mod_proxy_balancer.c>\n")
	sb.WriteString(fmt.Sprintf("    <Proxy balancer://%s>\n", upstream.Name))

	// 服务器列表
	for addr, options := range upstream.Servers {
		sb.WriteString(fmt.Sprintf("        BalancerMember %s\n", lo.If(options != "", addr+" "+options).Else(addr)))
	}

	// 负载均衡方法
	sb.WriteString(fmt.Sprintf("        ProxySet lbmethod=%s\n", lo.If(upstream.Algo != "", upstream.Algo).Else("byrequests")))

	// 连接池配置
	if upstream.Keepalive > 0 {
		sb.WriteString(fmt.Sprintf("        ProxySet max=%d\n", upstream.Keepalive))
	}

	sb.WriteString("    </Proxy>\n")
	sb.WriteString("</IfModule>\n")

	return sb.String()
}
