package frp

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/acepanel/panel/v3/internal/app"
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
	if err := os.MkdirAll(confDDir(), 0755); err != nil {
		return err
	}

	raw, err := toml.Marshal(confD)
	if err != nil {
		return err
	}

	return io.Write(itemPath(prefix, name), string(raw), 0644)
}
