package apache

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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

		redirect, err := parseRedirectFile(filepath.Join(siteDir, entry.Name()))
		if err != nil {
			continue // 跳过解析失败的文件
		}
		if redirect != nil {
			redirects = append(redirects, *redirect)
		}
	}

	return redirects, nil
}

// parseRedirectFile 解析单个重定向配置文件为结构体（基于 AST 遍历）
func parseRedirectFile(filePath string) (*types.Redirect, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseFragment(string(content))
	if err != nil {
		return nil, err
	}

	// Redirect 308 /old /new
	if d := cfg.FindOne("Redirect"); d != nil {
		vals := argValues(d.Args)
		if len(vals) >= 3 {
			code, _ := strconv.Atoi(vals[0])
			return &types.Redirect{
				Type:       types.RedirectTypeURL,
				From:       vals[1],
				To:         vals[2],
				StatusCode: code,
			}, nil
		}
	}

	// RedirectMatch 308 ^/old(.*)$ /new$1
	if d := cfg.FindOne("RedirectMatch"); d != nil {
		vals := argValues(d.Args)
		if len(vals) >= 3 {
			code, _ := strconv.Atoi(vals[0])
			to := vals[2]
			keepURI := strings.Contains(to, "$1")
			to = strings.TrimSuffix(to, "$1")
			from := strings.TrimPrefix(vals[1], "^")
			from = strings.TrimSuffix(from, "(.*)$")
			from = strings.TrimSuffix(from, "$")
			return &types.Redirect{
				Type:       types.RedirectTypeURL,
				From:       from,
				To:         to,
				KeepURI:    keepURI,
				StatusCode: code,
			}, nil
		}
	}

	// RewriteCond %{HTTP_HOST} ^host$ [NC] + RewriteRule ^(.*)$ to$1 [R=308,L]
	cond := cfg.FindOne("RewriteCond")
	rule := cfg.FindOne("RewriteRule")
	if cond != nil && rule != nil {
		condVals := argValues(cond.Args)
		ruleVals := argValues(rule.Args)
		if len(condVals) >= 2 && len(ruleVals) >= 2 {
			host := strings.TrimPrefix(condVals[1], "^")
			host = strings.TrimSuffix(host, "$")
			host = strings.ReplaceAll(host, `\.`, ".")
			to := ruleVals[1]
			keepURI := strings.Contains(to, "$1")
			to = strings.TrimSuffix(to, "$1")
			code := 308
			for _, v := range ruleVals[2:] {
				if c := parseRewriteStatus(v); c > 0 {
					code = c
				}
			}
			return &types.Redirect{
				Type:       types.RedirectTypeHost,
				From:       host,
				To:         to,
				KeepURI:    keepURI,
				StatusCode: code,
			}, nil
		}
	}

	// ErrorDocument 404 /custom
	if d := cfg.FindOne("ErrorDocument"); d != nil {
		vals := argValues(d.Args)
		if len(vals) >= 2 && vals[0] == "404" {
			return &types.Redirect{
				Type:       types.RedirectType404,
				To:         vals[1],
				StatusCode: 308,
			}, nil
		}
	}

	return nil, nil
}

// parseRewriteStatus 从 RewriteRule 的 flag（如 [R=308,L]）提取状态码
func parseRewriteStatus(flag string) int {
	for part := range strings.SplitSeq(strings.Trim(flag, "[]"), ",") {
		if rest, ok := strings.CutPrefix(part, "R="); ok {
			code, _ := strconv.Atoi(rest)
			return code
		}
	}
	return 0
}

// writeRedirectFiles 将重定向配置写入文件
func writeRedirectFiles(siteDir string, redirects []types.Redirect) error {
	if err := clearRedirectFiles(siteDir); err != nil {
		return err
	}

	for i, redirect := range redirects {
		num := RedirectStartNum + i
		if num > RedirectEndNum {
			return fmt.Errorf("redirect rules exceed limit (%d)", RedirectEndNum-RedirectStartNum+1)
		}

		filePath := filepath.Join(siteDir, fmt.Sprintf("%03d-redirect.conf", num))
		if err := os.WriteFile(filePath, []byte(generateRedirectConfig(redirect)), 0600); err != nil {
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
			if err := os.Remove(filepath.Join(siteDir, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete redirect config: %w", err)
			}
		}
	}

	return nil
}

// generateRedirectConfig 构建重定向配置 AST 并序列化
func generateRedirectConfig(redirect types.Redirect) string {
	statusCode := redirect.StatusCode
	if statusCode == 0 {
		statusCode = 308
	}

	cfg := &Config{}
	cfg.Append(Cmt("Auto-generated by AcePanel. DO NOT EDIT MANUALLY!"))

	switch redirect.Type {
	case types.RedirectTypeURL:
		cfg.Append(Cmt(fmt.Sprintf("URL redirect: %s -> %s", redirect.From, redirect.To)))
		if redirect.KeepURI {
			from := redirect.From
			if !strings.HasPrefix(from, "^") {
				from = "^" + from
			}
			if !strings.HasSuffix(from, "(.*)$") && !strings.HasSuffix(from, "$") {
				from += "(.*)$"
			}
			to := redirect.To
			if !strings.HasSuffix(to, "$1") {
				to += "$1"
			}
			cfg.Append(Dir("RedirectMatch", strconv.Itoa(statusCode), from, to))
		} else {
			cfg.Append(Dir("Redirect", strconv.Itoa(statusCode), redirect.From, redirect.To))
		}

	case types.RedirectTypeHost:
		cfg.Append(Cmt(fmt.Sprintf("Host redirect: %s -> %s", redirect.From, redirect.To)))
		cfg.Append(Dir("RewriteEngine", "on"))
		escapedHost := strings.ReplaceAll(redirect.From, ".", `\.`)
		cfg.Append(Dir("RewriteCond", "%{HTTP_HOST}", "^"+escapedHost+"$", "[NC]"))
		to := redirect.To
		if redirect.KeepURI {
			to += "$1"
		}
		cfg.Append(Dir("RewriteRule", "^(.*)$", to, fmt.Sprintf("[R=%d,L]", statusCode)))

	case types.RedirectType404:
		cfg.Append(Cmt(fmt.Sprintf("404 redirect -> %s", redirect.To)))
		cfg.Append(Dir("ErrorDocument", "404", redirect.To))
	}

	return cfg.Export() + "\n"
}
