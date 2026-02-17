package scan

import (
	"net"
	"time"
)

// Event 扫描事件
type Event struct {
	SourceIP  string
	Port      uint16
	Protocol  string // "tcp" / "udp"
	Timestamp time.Time
}

// InterfaceInfo 网卡信息
type InterfaceInfo struct {
	Name   string   `json:"name"`
	IPs    []string `json:"ips"`
	Status string   `json:"status"` // "up" / "down"
}

// ListInterfaces 列出可用网卡
func ListInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		info := InterfaceInfo{
			Name: iface.Name,
		}

		if iface.Flags&net.FlagUp != 0 {
			info.Status = "up"
		} else {
			info.Status = "down"
		}

		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.IPs = append(info.IPs, addr.String())
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// DefaultInterface 获取默认网卡名称
func DefaultInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		return iface.Name
	}

	return ""
}
