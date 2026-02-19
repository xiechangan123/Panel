package os

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/acepanel/panel/pkg/shell"
)

func readOSRelease() map[string]string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	osRelease := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := parts[0]
			value := strings.Trim(parts[1], `"`)
			osRelease[key] = value
		}
	}
	return osRelease
}

// IsDebian 判断是否是 Debian 系统
func IsDebian() bool {
	osRelease := readOSRelease()
	if osRelease == nil {
		return false
	}
	id, idLike := osRelease["ID"], osRelease["ID_LIKE"]
	return id == "debian" || strings.Contains(idLike, "debian")
}

// IsRHEL 判断是否是 RHEL 系统
func IsRHEL() bool {
	osRelease := readOSRelease()
	if osRelease == nil {
		return false
	}
	// alinux Alibaba Cloud Linux
	// hce Huawei Cloud EulerOS
	// openEuler openEuler
	id, idLike := osRelease["ID"], osRelease["ID_LIKE"]
	return id == "rhel" || id == "almalinux" || id == "rocky" || id == "alinux" || id == "tencentos" || id == "opencloudos" || strings.Contains(idLike, "rhel")
}

// IsUbuntu 判断是否是 Ubuntu 系统
func IsUbuntu() bool {
	osRelease := readOSRelease()
	if osRelease == nil {
		return false
	}
	id, idLike := osRelease["ID"], osRelease["ID_LIKE"]
	return id == "ubuntu" || strings.Contains(idLike, "ubuntu")
}

// IsEOL 判断系统是否已到达生命周期终点
func IsEOL() bool {
	eolTimeTable := map[string]map[string]time.Time{
		"rhel": {
			"9":  time.Date(2032, 5, 31, 0, 0, 0, 0, time.UTC),
			"10": time.Date(2035, 5, 31, 0, 0, 0, 0, time.UTC),
		},
		"debian": {
			"12": time.Date(2028, 6, 30, 0, 0, 0, 0, time.UTC),
			"13": time.Date(2030, 6, 30, 0, 0, 0, 0, time.UTC),
		},
		"ubuntu": {
			"22.04": time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC),
			"24.04": time.Date(2029, 5, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	osRelease := readOSRelease()

	if IsRHEL() {
		version, ok := osRelease["VERSION_ID"]
		if !ok {
			return false
		}
		majorVersion := strings.Split(version, ".")[0]
		if eol, ok := eolTimeTable["rhel"][majorVersion]; ok {
			return time.Now().After(eol)
		}
	}
	if IsUbuntu() {
		version, ok := osRelease["VERSION_ID"]
		if !ok {
			return false
		}
		majorVersion := strings.Join(strings.Split(version, ".")[:2], ".")
		if eol, ok := eolTimeTable["ubuntu"][majorVersion]; ok {
			return time.Now().After(eol)
		}
	}
	if IsDebian() {
		version, ok := osRelease["VERSION_ID"]
		if !ok {
			return false
		}
		majorVersion := strings.Split(version, ".")[0]
		if eol, ok := eolTimeTable["debian"][majorVersion]; ok {
			return time.Now().After(eol)
		}
	}

	return false
}

func TCPPortInUse(port uint) bool {
	addr := fmt.Sprintf(":%d", port)
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	defer func(conn net.Listener) { _ = conn.Close() }(conn)
	return false
}

func UDPPortInUse(port uint) bool {
	addr := fmt.Sprintf(":%d", port)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return true
	}
	defer func(conn net.PacketConn) { _ = conn.Close() }(conn)
	return false
}

// PortProcess 端口占用进程信息
type PortProcess struct {
	PID     string `json:"pid"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

// GetPortProcess 获取占用指定端口的进程信息
func GetPortProcess(port uint, protocol string) []PortProcess {
	var flag string
	switch protocol {
	case "udp":
		flag = "-ulnp"
	default:
		flag = "-tlnp"
	}

	output, err := shell.Execf("ss %s sport = :%d", flag, port)
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`\("([^"]*)",pid=(\d+),`)
	seen := make(map[string]struct{})
	var processes []PortProcess

	for line := range strings.SplitSeq(output, "\n") {
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			pid := match[2]
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}

			name := match[1]
			command := name
			if cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%s/cmdline", pid)); err == nil {
				// cmdline 以 \0 分隔参数
				cmd := strings.ReplaceAll(strings.TrimRight(string(cmdline), "\x00"), "\x00", " ")
				if cmd != "" {
					command = cmd
				}
			}

			processes = append(processes, PortProcess{
				PID:     pid,
				Name:    name,
				Command: command,
			})
		}
	}

	return processes
}
