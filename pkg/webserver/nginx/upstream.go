package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/acepanel/panel/pkg/webserver/types"
)

// upstreamFilePattern 匹配 upstream 配置文件名 (100-XXX-name.conf)
var upstreamFilePattern = regexp.MustCompile(`^(\d{3})-(.+)\.conf$`)

// parseUpstreamFiles 从 shared 目录解析所有 upstream 配置
func parseUpstreamFiles(sharedDir string) (map[string]types.Upstream, error) {
	entries, err := os.ReadDir(sharedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	upstreams := make(map[string]types.Upstream)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := upstreamFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num < UpstreamStartNum {
			continue
		}

		name := matches[2]
		filePath := filepath.Join(sharedDir, entry.Name())
		upstream, err := parseUpstreamFile(filePath, name)
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if upstream != nil {
			upstreams[name] = *upstream
		}
	}

	return upstreams, nil
}

// parseUpstreamFile 解析单个 upstream 配置文件
func parseUpstreamFile(filePath string, expectedName string) (*types.Upstream, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	// 解析 upstream 块
	// upstream backend {
	//     least_conn;
	//     server 127.0.0.1:8080 weight=5;
	//     keepalive 32;
	// }
	upstreamPattern := regexp.MustCompile(`upstream\s+(\S+)\s*\{([^}]+)}`)
	matches := upstreamPattern.FindStringSubmatch(contentStr)
	if matches == nil {
		return nil, nil
	}

	name := matches[1]
	if expectedName != "" && name != expectedName {
		return nil, nil
	}

	blockContent := matches[2]
	upstream := &types.Upstream{
		Servers: make(map[string]string),
	}

	// 解析负载均衡算法
	algoPatterns := []string{"least_conn", "ip_hash", "hash", "random"}
	for _, algo := range algoPatterns {
		if regexp.MustCompile(`\b` + algo + `\b`).MatchString(blockContent) {
			upstream.Algo = algo
			break
		}
	}

	// 解析 server 指令
	serverPattern := regexp.MustCompile(`server\s+(\S+)(?:\s+([^;]+))?;`)
	serverMatches := serverPattern.FindAllStringSubmatch(blockContent, -1)
	for _, sm := range serverMatches {
		addr := sm[1]
		options := ""
		if len(sm) > 2 {
			options = strings.TrimSpace(sm[2])
		}
		upstream.Servers[addr] = options
	}

	// 解析 keepalive 指令
	keepalivePattern := regexp.MustCompile(`keepalive\s+(\d+);`)
	if km := keepalivePattern.FindStringSubmatch(blockContent); km != nil {
		upstream.Keepalive, _ = strconv.Atoi(km[1])
	}

	return upstream, nil
}

// writeUpstreamFiles 将 upstream 配置写入文件
func writeUpstreamFiles(sharedDir string, upstreams map[string]types.Upstream) error {
	// 删除现有的 upstream 配置文件
	if err := clearUpstreamFiles(sharedDir); err != nil {
		return err
	}

	// 写入新的配置文件
	num := UpstreamStartNum
	for name, upstream := range upstreams {
		fileName := fmt.Sprintf("%03d-%s.conf", num, name)
		filePath := filepath.Join(sharedDir, fileName)

		content := generateUpstreamConfig(name, upstream)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write upstream config: %w", err)
		}
		num++
	}

	return nil
}

// clearUpstreamFiles 清除所有 upstream 配置文件
func clearUpstreamFiles(sharedDir string) error {
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

		matches := upstreamFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num >= UpstreamStartNum {
			filePath := filepath.Join(sharedDir, entry.Name())
			if err = os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete upstream config: %w", err)
			}
		}
	}

	return nil
}

// generateUpstreamConfig 生成 upstream 配置内容
func generateUpstreamConfig(name string, upstream types.Upstream) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Upstream: %s\n", name))
	sb.WriteString(fmt.Sprintf("upstream %s {\n", name))

	// 负载均衡算法
	if upstream.Algo != "" {
		sb.WriteString(fmt.Sprintf("    %s;\n", upstream.Algo))
	}

	// 服务器列表
	for addr, options := range upstream.Servers {
		if options != "" {
			sb.WriteString(fmt.Sprintf("    server %s %s;\n", addr, options))
		} else {
			sb.WriteString(fmt.Sprintf("    server %s;\n", addr))
		}
	}

	// keepalive 连接数
	if upstream.Keepalive > 0 {
		sb.WriteString(fmt.Sprintf("    keepalive %d;\n", upstream.Keepalive))
	}

	sb.WriteString("}\n")

	return sb.String()
}
