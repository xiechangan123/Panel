package nginx

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// AuthConfName 基本认证 map 片段文件名
const AuthConfName = "011-auth.conf"

// 基本认证 map 变量名前缀
const (
	authRealmVarPrefix = "$ace_auth_realm_"
	authFileVarPrefix  = "$ace_auth_file_"
)

// SafeName 将名称转换为 nginx 标识符安全形式
func SafeName(name string) string {
	return strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			return char
		}
		return '_'
	}, name)
}

// authVarNames 返回站点专属的 realm/file map 变量名
func authVarNames(siteName string) (string, string) {
	suffix := SafeName(siteName)
	return authRealmVarPrefix + suffix, authFileVarPrefix + suffix
}

// authPathPattern 将路径转换为 map 正则，如 /admin -> ~^/admin(/.*)?$
func authPathPattern(path string) string {
	return "~^" + regexp.QuoteMeta(path) + "(/.*)?$"
}

// authPatternPath 将 map 正则还原为路径
func authPatternPath(pattern string) string {
	path := strings.TrimPrefix(pattern, "~^")
	path = strings.TrimSuffix(path, "(/.*)?$")
	return strings.ReplaceAll(path, `\`, "")
}

// generateAuthMaps 生成 realm/file 两个 map 块，通过 $uri 匹配实现目录级认证，
// 从而对静态、PHP、反向代理的所有 location 统一生效（auth_basic 变量值为 off 时禁用认证）
func generateAuthMaps(siteName string, auths []types.BasicAuth) string {
	realmVar, fileVar := authVarNames(siteName)
	realmDefault, fileDefault := "off", `""`
	var dirs []types.BasicAuth
	for _, auth := range auths {
		if auth.Path == "/" {
			realmDefault = `"Restricted"`
			fileDefault = fmt.Sprintf("%q", auth.UserFile)
			continue
		}
		dirs = append(dirs, auth)
	}
	// map 按声明顺序取首个命中的正则，更长（更精确）的路径需排在前面
	slices.SortStableFunc(dirs, func(a, b types.BasicAuth) int {
		return len(b.Path) - len(a.Path)
	})

	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "map $uri %s {\n    default %s;\n", realmVar, realmDefault)
	for _, dir := range dirs {
		_, _ = fmt.Fprintf(&sb, "    %q \"Restricted\";\n", authPathPattern(dir.Path))
	}
	sb.WriteString("}\n")
	_, _ = fmt.Fprintf(&sb, "map $uri %s {\n    default %s;\n", fileVar, fileDefault)
	for _, dir := range dirs {
		_, _ = fmt.Fprintf(&sb, "    %q %q;\n", authPathPattern(dir.Path), dir.UserFile)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// parseAuthMaps 从片段内容中解析 file map 还原认证规则
func parseAuthMaps(content string) []types.BasicAuth {
	// 截取 file map 块体（片段由本包生成，file map 唯一且以 } 结束）
	_, rest, ok := strings.Cut(content, "map $uri "+authFileVarPrefix)
	if !ok {
		return nil
	}
	body, _, _ := strings.Cut(rest, "}")

	var auths []types.BasicAuth
	for _, line := range strings.Split(body, "\n")[1:] {
		key, value, ok := strings.Cut(strings.TrimSuffix(strings.TrimSpace(line), ";"), " ")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		if value == "" {
			continue
		}
		if key == "default" {
			auths = append(auths, types.BasicAuth{Path: "/", UserFile: value})
		} else {
			auths = append(auths, types.BasicAuth{Path: authPatternPath(strings.Trim(key, `"`)), UserFile: value})
		}
	}
	return auths
}
