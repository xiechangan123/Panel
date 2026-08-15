package rsync

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/pkg/io"
)

const (
	rsyncdConf = "/etc/rsyncd.conf"
	// rsyncdDir rsyncd 原生的 include 目录，只会读取其中的 *.conf
	rsyncdDir = "/etc/rsyncd.d"
	// includeDirective 指令必须落在全局段内，即首个 [module] 之前
	includeDirective = "&include " + rsyncdDir
)

func modulePath(name string) string {
	return filepath.Join(rsyncdDir, name+".conf")
}

// secretsPath 一个模块独立的密钥文件，.secrets 后缀不会被 rsyncd 当作模块读入
func secretsPath(name string) string {
	return filepath.Join(rsyncdDir, name+".secrets")
}

// ensureInclude 保证 rsyncd.d 目录存在且被 rsyncd.conf 的全局段引用
func ensureInclude() error {
	// io.Write 拿文件权限当目录权限，这里要显式建目录才有 x 位
	if err := os.MkdirAll(rsyncdDir, 0755); err != nil {
		return err
	}

	config, err := io.Read(rsyncdConf)
	if err != nil {
		return err
	}
	if strings.Contains(config, includeDirective) {
		return nil
	}

	// 插到首个 [module] 之前，否则会继承上一个模块的参数而非全局默认值
	raws := strings.Split(config, "\n")
	idx := slices.IndexFunc(raws, func(raw string) bool { return strings.HasPrefix(strings.TrimSpace(raw), "[") })
	if idx < 0 {
		idx = len(raws)
	}

	return io.Write(rsyncdConf, strings.Join(slices.Insert(raws, idx, includeDirective), "\n"), 0644)
}

func readModule(name string) (Module, error) {
	config, err := io.Read(modulePath(name))
	if err != nil {
		return Module{}, err
	}

	secret, _ := io.Read(secretsPath(name))
	_, secret, _ = strings.Cut(strings.TrimSpace(secret), ":")

	return Module{
		Name:       name,
		Path:       confval.SectionINI.GetIn(config, name, "path"),
		Comment:    confval.SectionINI.GetIn(config, name, "comment"),
		ReadOnly:   confval.SectionINI.GetIn(config, name, "read only") == "yes",
		AuthUser:   confval.SectionINI.GetIn(config, name, "auth users"),
		HostsAllow: confval.SectionINI.GetIn(config, name, "hosts allow"),
		Secret:     secret,
	}, nil
}

func listModules() ([]Module, error) {
	// Glob 在目录不存在时返回空，无需额外判断
	paths, err := filepath.Glob(modulePath("*"))
	if err != nil {
		return nil, err
	}

	modules := make([]Module, 0, len(paths))
	for _, path := range paths {
		module, err := readModule(strings.TrimSuffix(filepath.Base(path), ".conf"))
		if err != nil {
			return nil, err
		}

		modules = append(modules, module)
	}

	return modules, nil
}

func writeModule(module Module) error {
	if err := ensureInclude(); err != nil {
		return err
	}

	config := fmt.Sprintf(
		"[%s]\npath = %s\ncomment = %s\nread only = %s\nauth users = %s\nhosts allow = %s\nsecrets file = %s\n",
		module.Name, module.Path, module.Comment, lo.Ternary(module.ReadOnly, "yes", "no"),
		module.AuthUser, module.HostsAllow, secretsPath(module.Name),
	)
	if err := io.Write(modulePath(module.Name), config, 0644); err != nil {
		return err
	}

	return io.Write(secretsPath(module.Name), module.AuthUser+":"+module.Secret+"\n", 0600)
}
