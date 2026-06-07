package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/tufanbarisyildirim/gonginx/config"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// redirectFilePattern 匹配重定向配置文件名 (100-199)
var redirectFilePattern = regexp.MustCompile(`^(\d{3})-redirect\.conf$`)

// parseRedirectFiles 从 site 目录解析所有重定向配置
func parseRedirectFiles(siteDir string) ([]types.Redirect, error) {
	entries, err := os.ReadDir(siteDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var redirects []types.Redirect
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := redirectFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num < RedirectStartNum || num > RedirectEndNum {
			continue
		}

		filePath := filepath.Join(siteDir, entry.Name())
		redirect, err := parseRedirectFile(filePath)
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if redirect != nil {
			redirects = append(redirects, *redirect)
		}
	}

	return redirects, nil
}

// parseRedirectFile 解析单个重定向配置文件
func parseRedirectFile(filePath string) (*types.Redirect, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	p, err := NewParserFromString(string(content))
	if err != nil {
		return nil, err
	}
	cfg := p.Config()

	// 取 return 指令的状态码与目标（三种重定向各只含一条 return）
	statusCode, to, keepURI, ok := parseReturn(p, cfg.FindDirectives("return"))
	if !ok {
		return nil, nil
	}

	// 解析 Host 重定向: if ($host = "old.example.com") { return 308 https://new.example.com$request_uri; }
	for _, ifd := range cfg.FindDirectives("if") {
		params := p.parameters2Slices(ifd.GetParameters())
		// gonginx 形态: ["($host", "=", "\"old.example.com\"", ")"]
		if len(params) >= 3 && strings.Contains(params[0], "$host") {
			return &types.Redirect{
				Type:       types.RedirectTypeHost,
				From:       unquote(strings.TrimSuffix(params[2], ")")),
				To:         to,
				KeepURI:    keepURI,
				StatusCode: statusCode,
			}, nil
		}
	}

	// 解析 404 重定向: error_page 404 = @redirect_404; location @redirect_404 { return 308 /custom; }
	for _, ed := range cfg.FindDirectives("error_page") {
		params := p.parameters2Slices(ed.GetParameters())
		if len(params) > 0 && params[0] == "404" {
			return &types.Redirect{
				Type:       types.RedirectType404,
				From:       "",
				To:         to,
				KeepURI:    keepURI,
				StatusCode: statusCode,
			}, nil
		}
	}

	// 解析 URL 重定向: location = /old { return 308 /new; }
	for _, ld := range cfg.FindDirectives("location") {
		params := p.parameters2Slices(ld.GetParameters())
		// 带 = 修饰符的 location，形态: ["=", "/old"]
		if len(params) >= 2 && params[0] == "=" {
			return &types.Redirect{
				Type:       types.RedirectTypeURL,
				From:       params[1],
				To:         to,
				KeepURI:    keepURI,
				StatusCode: statusCode,
			}, nil
		}
	}

	return nil, nil
}

// parseReturn 从 return 指令解析状态码、目标与是否保留 URI
func parseReturn(p *Parser, dirs []config.IDirective) (statusCode int, to string, keepURI bool, ok bool) {
	if len(dirs) == 0 {
		return 0, "", false, false
	}
	params := p.parameters2Slices(dirs[0].GetParameters())
	if len(params) < 2 {
		return 0, "", false, false
	}
	statusCode, _ = strconv.Atoi(params[0])
	to = strings.Join(params[1:], " ")
	keepURI = strings.Contains(to, "$request_uri")
	if keepURI {
		to = strings.TrimSuffix(to, "$request_uri")
	}
	return statusCode, to, keepURI, true
}

// writeRedirectFiles 将重定向配置写入文件
func writeRedirectFiles(siteDir string, redirects []types.Redirect) error {
	// 删除现有的重定向配置文件 (100-199)
	if err := clearRedirectFiles(siteDir); err != nil {
		return err
	}

	// 写入新的配置文件
	for i, redirect := range redirects {
		num := RedirectStartNum + i
		if num > RedirectEndNum {
			return fmt.Errorf("redirect rules exceed limit (%d)", RedirectEndNum-RedirectStartNum+1)
		}

		fileName := fmt.Sprintf("%03d-redirect.conf", num)
		filePath := filepath.Join(siteDir, fileName)

		content := generateRedirectConfig(redirect)
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write redirect config: %w", err)
		}
	}

	return nil
}

// clearRedirectFiles 清除所有重定向配置文件
func clearRedirectFiles(siteDir string) error {
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

		matches := redirectFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		num, _ := strconv.Atoi(matches[1])
		if num >= RedirectStartNum && num <= RedirectEndNum {
			filePath := filepath.Join(siteDir, entry.Name())
			if err = os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete redirect config: %w", err)
			}
		}
	}

	return nil
}

// generateRedirectConfig 生成重定向配置内容
func generateRedirectConfig(redirect types.Redirect) string {
	statusCode := lo.If(redirect.StatusCode == 0, 308).Else(redirect.StatusCode)
	uriSuffix := lo.If(redirect.KeepURI, "$request_uri").Else("")

	var sb strings.Builder
	sb.WriteString("# Auto-generated by AcePanel. DO NOT EDIT MANUALLY!\n")

	switch redirect.Type {
	case types.RedirectTypeURL:
		// URL 重定向
		_, _ = fmt.Fprintf(&sb, "# URL redirect: %s -> %s\n", redirect.From, redirect.To)
		_, _ = fmt.Fprintf(&sb, "location = %s {\n", redirect.From)
		_, _ = fmt.Fprintf(&sb, "    return %d %s%s;\n", statusCode, redirect.To, uriSuffix)
		sb.WriteString("}\n")

	case types.RedirectTypeHost:
		// Host 重定向
		_, _ = fmt.Fprintf(&sb, "# Host redirect: %s -> %s\n", redirect.From, redirect.To)
		_, _ = fmt.Fprintf(&sb, "if ($host = \"%s\") {\n", redirect.From)
		_, _ = fmt.Fprintf(&sb, "    return %d %s%s;\n", statusCode, redirect.To, uriSuffix)
		sb.WriteString("}\n")

	case types.RedirectType404:
		// 404 重定向
		_, _ = fmt.Fprintf(&sb, "# 404 redirect -> %s\n", redirect.To)
		sb.WriteString("error_page 404 = @redirect_404;\n")
		sb.WriteString("location @redirect_404 {\n")
		_, _ = fmt.Fprintf(&sb, "    return %d %s%s;\n", statusCode, redirect.To, uriSuffix)
		sb.WriteString("}\n")
	}

	return sb.String()
}
