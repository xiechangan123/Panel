package frp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/pkg/io"
)

const (
	proxyPrefix   = "proxy"
	visitorPrefix = "visitor"
)

func confPath(name string) string {
	return filepath.Join(app.Root, "server", "frp", name+".toml")
}

func confDDir() string {
	return filepath.Join(app.Root, "server", "frp", "conf.d")
}

func confDGlob() string {
	return filepath.Join(confDDir(), "*.toml")
}

func itemPath(prefix, name string) string {
	return filepath.Join(confDDir(), prefix+"-"+name+".toml")
}

// ensureConfD 保证 conf.d 目录存在且已被 frpc.toml 的 includes 引用
// 目录缺失会让 frpc 直接启动失败，因此每次写操作前都要调用
func ensureConfD() error {
	// io.Write 拿文件权限当目录权限，这里要显式建目录才有 x 位
	if err := os.MkdirAll(confDDir(), 0755); err != nil {
		return err
	}

	path := confPath("frpc")
	config, err := io.Read(path)
	if err != nil {
		return err
	}

	includes, err := parseIncludes(confval.GetTOML(config, "includes"))
	if err != nil {
		return err
	}

	glob := confDGlob()
	if slices.Contains(includes, glob) {
		return nil
	}

	return io.Write(path, confval.SetTOML(config, "includes", append(includes, glob)), 0644)
}

// parseIncludes 解析不了说明是跨行数组等改不动的写法，直接报错而不是覆盖掉用户的配置
func parseIncludes(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}

	var includes []string
	if err := toml.Unmarshal([]byte("includes = "+raw), &struct {
		Includes *[]string `toml:"includes"`
	}{Includes: &includes}); err != nil {
		return nil, fmt.Errorf("failed to parse includes in frpc.toml, please write it on a single line: %w", err)
	}

	return includes, nil
}

func readConfD(path string) (ConfD, error) {
	raw, err := io.Read(path)
	if err != nil {
		return ConfD{}, err
	}

	var confD ConfD
	return confD, toml.Unmarshal([]byte(raw), &confD)
}

func listConfD() (ConfD, error) {
	var all ConfD

	// Glob 在目录不存在时返回空，无需额外判断
	paths, err := filepath.Glob(confDGlob())
	if err != nil {
		return all, err
	}

	for _, path := range paths {
		confD, err := readConfD(path)
		if err != nil {
			return all, err
		}

		all.Proxies = append(all.Proxies, confD.Proxies...)
		all.Visitors = append(all.Visitors, confD.Visitors...)
	}

	return all, nil
}

func writeConfD(prefix, name string, confD ConfD) error {
	if err := ensureConfD(); err != nil {
		return err
	}

	raw, err := toml.Marshal(confD)
	if err != nil {
		return err
	}

	return io.Write(itemPath(prefix, name), string(raw), 0644)
}
