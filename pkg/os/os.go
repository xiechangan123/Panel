package os

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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
