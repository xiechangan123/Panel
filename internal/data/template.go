package data

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/api"
	"github.com/acepanel/panel/pkg/firewall"
	"github.com/acepanel/panel/pkg/types"
)

type templateRepo struct {
	api      *api.API
	firewall *firewall.Firewall
}

func NewTemplateRepo() biz.TemplateRepo {
	return &templateRepo{
		api:      api.NewAPI(app.Version, app.Locale),
		firewall: firewall.NewFirewall(),
	}
}

// List 获取所有模版
func (r *templateRepo) List() (api.Templates, error) {
	templates, err := r.api.Templates()
	if err != nil {
		return nil, err
	}
	return *templates, nil
}

// Get 获取模版详情
func (r *templateRepo) Get(slug string) (*api.Template, error) {
	templates, err := r.api.Templates()
	if err != nil {
		return nil, err
	}

	for _, t := range *templates {
		if t.Slug == slug {
			return t, nil
		}
	}

	return nil, fmt.Errorf("template %s not found", slug)
}

// Callback 模版下载回调
func (r *templateRepo) Callback(slug string) error {
	return r.api.TemplateCallback(slug)
}

// CreateCompose 创建编排
func (r *templateRepo) CreateCompose(name, compose string, envs []types.KV, autoFirewall bool) error {
	dir := filepath.Join(app.Root, "server", "compose", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		return err
	}

	var sb strings.Builder
	for _, kv := range envs {
		sb.WriteString(kv.Key)
		sb.WriteString("=")
		sb.WriteString(kv.Value)
		sb.WriteString("\n")
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(sb.String()), 0644); err != nil {
		return err
	}

	// 自动放行端口
	if autoFirewall {
		ports := r.parsePortsFromCompose(compose)
		for _, port := range ports {
			_ = r.firewall.Port(firewall.FireInfo{
				Family:    "ipv4",
				PortStart: port.Port,
				PortEnd:   port.Port,
				Protocol:  port.Protocol,
				Strategy:  firewall.StrategyAccept,
				Direction: "in",
			}, firewall.OperationAdd)
		}
	}

	return nil
}

type composePort struct {
	Port     uint
	Protocol firewall.Protocol
}

// parsePortsFromCompose 从 compose 文件中解析端口
func (r *templateRepo) parsePortsFromCompose(compose string) []composePort {
	var ports []composePort
	seen := make(map[string]bool)

	// 匹配 ports 部分的端口映射
	// 支持格式: "8080:80", "8080:80/tcp", "8080:80/udp", "80", "80/tcp"
	portRegex := regexp.MustCompile(`(?m)^\s*-\s*["']?(\d+)(?::\d+)?(?:/(\w+))?["']?\s*$`)
	matches := portRegex.FindAllStringSubmatch(compose, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		portStr := match[1]
		protocol := firewall.ProtocolTCP
		if len(match) > 2 && match[2] != "" {
			switch strings.ToLower(match[2]) {
			case "udp":
				protocol = firewall.ProtocolUDP
			case "tcp":
				protocol = firewall.ProtocolTCP
			}
		}

		// 去重
		key := portStr + "/" + string(protocol)
		if seen[key] {
			continue
		}
		seen[key] = true

		var port uint
		if _, _, found := strings.Cut(portStr, ":"); found {
			// 格式: host:container
			parts := strings.Split(portStr, ":")
			if len(parts) > 0 {
				port = parseUint(parts[0])
			}
		} else {
			port = parseUint(portStr)
		}

		if port > 0 && port <= 65535 {
			ports = append(ports, composePort{
				Port:     port,
				Protocol: protocol,
			})
		}
	}

	return ports
}

func parseUint(s string) uint {
	var n uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint(c-'0')
		} else {
			break
		}
	}
	return n
}
