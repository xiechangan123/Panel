package frp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/pkg/io"
)

// confPath frps.toml 或 frpc.toml 的路径
func confPath(name string) string {
	return filepath.Join(app.Root, "server", "frp", name+".toml")
}

// confDDir 面板管理的代理与访问者所在目录
func confDDir() string {
	return filepath.Join(app.Root, "server", "frp", "conf.d")
}

// confDGlob frpc.toml 里 includes 使用的匹配串
func confDGlob() string {
	return filepath.Join(confDDir(), "*.toml")
}

// itemPath 一个代理或访问者对应的文件，分前缀避免同名代理与访问者撞文件
func itemPath(prefix, name string) string {
	return filepath.Join(confDDir(), prefix+"-"+name+".toml")
}

// ensureConfD 保证 conf.d 目录存在且已被 frpc.toml 的 includes 引用
func ensureConfD() error {
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

// parseIncludes 解析 includes 数组的原始字面量
// 解析不了说明是跨行数组等改不动的写法，直接报错而不是覆盖掉用户的配置
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

// listConfD 读出 conf.d 目录下的全部代理与访问者
func listConfD() (ConfD, error) {
	var all ConfD

	entries, err := os.ReadDir(confDDir())
	if err != nil {
		// 目录尚未创建时视为空
		if os.IsNotExist(err) {
			return all, nil
		}
		return all, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		raw, err := os.ReadFile(filepath.Join(confDDir(), entry.Name()))
		if err != nil {
			return all, err
		}

		var confD ConfD
		if err = toml.Unmarshal(raw, &confD); err != nil {
			return all, err
		}

		all.Proxies = append(all.Proxies, confD.Proxies...)
		all.Visitors = append(all.Visitors, confD.Visitors...)
	}

	return all, nil
}

// writeConfD 写入单个代理或访问者的配置文件
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
