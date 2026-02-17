//go:build linux

package scan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

const (
	ringBufSize = 256 * 1024 // 256KB
	eventSize   = 20         // src_ip(16) + dst_port(2) + protocol(1) + version(1)
)

// __sk_buff 字段偏移量
const (
	skbData    = 76 // __sk_buff.data
	skbDataEnd = 80 // __sk_buff.data_end
)

// 协议头长度
const (
	ethLen  = 14
	ipLen   = 20 // IPv4 最小头长度
	ipv6Len = 40 // IPv6 固定头长度
	tcpLen  = 20 // TCP 最小头长度
	udpLen  = 8  // UDP 头长度
)

// EtherType（小端序 16 位加载值）
const (
	ethIPv4 = 0x0008 // ETH_P_IP (0x0800)
	ethIPv6 = 0xDD86 // ETH_P_IPV6 (0x86DD)
)

// 协议常量
const (
	ipTCP      = 6
	ipUDP      = 17
	synAckMask = 0x12 // SYN + ACK
	synOnly    = 0x02 // 仅 SYN
)

// IPv4 字段偏移量（从 data 起始）
const (
	offV4Proto = ethLen + 9  // IP protocol (23)
	offV4SrcIP = ethLen + 12 // IP 源地址 (26)
)

// IPv6 字段偏移量（从 data 起始）
const (
	offV6Proto = ethLen + 6 // Next Header (20)
	offV6SrcIP = ethLen + 8 // 源地址 (22)，16 字节
)

// ifaceHandle 单个网卡的 eBPF 挂载句柄
type ifaceHandle struct {
	link link.Link
}

// Scanner eBPF 扫描检测器
type Scanner struct {
	prog      *ebpf.Program
	events    *ebpf.Map
	ports     *ebpf.Map // 监听端口白名单（hash map，命中则跳过）
	handles   map[string]*ifaceHandle
	reader    *ringbuf.Reader
	eventsCh  chan Event
	stopCh    chan struct{}
	mu        sync.Mutex
	log       *slog.Logger
	lastPorts map[uint16]bool
}

// newPortsMap 创建监听端口白名单 BPF map
func newPortsMap() (*ebpf.Map, error) {
	return ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.Hash,
		KeySize:    4, // uint32
		ValueSize:  1, // uint8
		MaxEntries: 65535,
	})
}

// buildDetector eBPF TC ingress 扫描检测程序
// 只捕获目标端口不在 ports 白名单中的 SYN/UDP 包
func buildDetector(events, ports *ebpf.Map) (*ebpf.Program, error) {
	// TCP 处理器：边界检查 → SYN 过滤 → 端口白名单 → 输出事件
	tcpHandler := func(sym string, boundsEnd, flagsOff, portOff int) asm.Instructions {
		return asm.Instructions{
			asm.Mov.Reg(asm.R2, asm.R6).WithSymbol(sym),
			asm.Add.Imm(asm.R2, int32(boundsEnd)),
			asm.JGT.Reg(asm.R2, asm.R7, "exit"),

			// 仅 SYN 包
			asm.LoadMem(asm.R2, asm.R6, int16(flagsOff), asm.Byte),
			asm.And.Imm(asm.R2, synAckMask),
			asm.JNE.Imm(asm.R2, synOnly, "exit"),

			// 加载端口并转换字节序
			asm.LoadMem(asm.R2, asm.R6, int16(portOff), asm.Half),
			asm.HostTo(asm.BE, asm.R2, asm.Half),
			asm.Mov.Reg(asm.R9, asm.R2), // 保存端口到 R9

			// 查询端口白名单
			asm.StoreMem(asm.RFP, -24, asm.R9, asm.Word),
			asm.LoadMapPtr(asm.R1, ports.FD()),
			asm.Mov.Reg(asm.R2, asm.RFP),
			asm.Add.Imm(asm.R2, -24),
			asm.FnMapLookupElem.Call(),
			asm.JNE.Imm(asm.R0, 0, "exit"),

			// 写入事件
			asm.StoreMem(asm.RFP, -4, asm.R9, asm.Half),
			asm.StoreImm(asm.RFP, -2, ipTCP, asm.Byte),

			asm.LoadMapPtr(asm.R1, events.FD()),
			asm.Mov.Reg(asm.R2, asm.RFP),
			asm.Add.Imm(asm.R2, -eventSize),
			asm.Mov.Imm(asm.R3, eventSize),
			asm.Mov.Imm(asm.R4, 0),
			asm.FnRingbufOutput.Call(),
			asm.Ja.Label("exit"),
		}
	}

	// UDP 处理器：边界检查 → 端口白名单 → 输出事件
	udpHandler := func(sym string, boundsEnd, portOff int) asm.Instructions {
		return asm.Instructions{
			asm.Mov.Reg(asm.R2, asm.R6).WithSymbol(sym),
			asm.Add.Imm(asm.R2, int32(boundsEnd)),
			asm.JGT.Reg(asm.R2, asm.R7, "exit"),

			asm.LoadMem(asm.R2, asm.R6, int16(portOff), asm.Half),
			asm.HostTo(asm.BE, asm.R2, asm.Half),
			asm.Mov.Reg(asm.R9, asm.R2),

			// 查询端口白名单
			asm.StoreMem(asm.RFP, -24, asm.R9, asm.Word),
			asm.LoadMapPtr(asm.R1, ports.FD()),
			asm.Mov.Reg(asm.R2, asm.RFP),
			asm.Add.Imm(asm.R2, -24),
			asm.FnMapLookupElem.Call(),
			asm.JNE.Imm(asm.R0, 0, "exit"),

			asm.StoreMem(asm.RFP, -4, asm.R9, asm.Half),
			asm.StoreImm(asm.RFP, -2, ipUDP, asm.Byte),

			asm.LoadMapPtr(asm.R1, events.FD()),
			asm.Mov.Reg(asm.R2, asm.RFP),
			asm.Add.Imm(asm.R2, -eventSize),
			asm.Mov.Imm(asm.R3, eventSize),
			asm.Mov.Imm(asm.R4, 0),
			asm.FnRingbufOutput.Call(),
			asm.Ja.Label("exit"),
		}
	}

	var insns asm.Instructions

	// 加载 skb->data / skb->data_end
	insns = append(insns,
		asm.LoadMem(asm.R6, asm.R1, skbData, asm.Word),
		asm.LoadMem(asm.R7, asm.R1, skbDataEnd, asm.Word),
	)

	// 边界检查：以太网头
	insns = append(insns,
		asm.Mov.Reg(asm.R2, asm.R6),
		asm.Add.Imm(asm.R2, ethLen),
		asm.JGT.Reg(asm.R2, asm.R7, "exit"),
	)

	// EtherType 分支
	insns = append(insns,
		asm.LoadMem(asm.R0, asm.R6, 12, asm.Half),
		asm.JEq.Imm(asm.R0, ethIPv4, "ipv4"),
		asm.JEq.Imm(asm.R0, ethIPv6, "ipv6"),
		asm.Ja.Label("exit"),
	)

	// ========== IPv4 ==========
	insns = append(insns,
		asm.Mov.Reg(asm.R2, asm.R6).WithSymbol("ipv4"),
		asm.Add.Imm(asm.R2, ethLen+ipLen),
		asm.JGT.Reg(asm.R2, asm.R7, "exit"),

		asm.LoadMem(asm.R8, asm.R6, offV4Proto, asm.Byte),
		asm.LoadMem(asm.R9, asm.R6, offV4SrcIP, asm.Word),

		// 栈布局：[src_ip(16) | port(2) | proto(1) | ver(1)] = 20 字节
		asm.StoreMem(asm.RFP, -20, asm.R9, asm.Word),
		asm.StoreImm(asm.RFP, -16, 0, asm.Word),
		asm.StoreImm(asm.RFP, -12, 0, asm.Word),
		asm.StoreImm(asm.RFP, -8, 0, asm.Word),
		asm.StoreImm(asm.RFP, -1, 4, asm.Byte), // version = 4

		asm.JEq.Imm(asm.R8, ipTCP, "v4tcp"),
		asm.JEq.Imm(asm.R8, ipUDP, "v4udp"),
		asm.Ja.Label("exit"),
	)

	v4t := ethLen + ipLen
	insns = append(insns, tcpHandler("v4tcp", v4t+tcpLen, v4t+13, v4t+2)...)
	insns = append(insns, udpHandler("v4udp", v4t+udpLen, v4t+2)...)

	// ========== IPv6 ==========
	insns = append(insns,
		asm.Mov.Reg(asm.R2, asm.R6).WithSymbol("ipv6"),
		asm.Add.Imm(asm.R2, ethLen+ipv6Len),
		asm.JGT.Reg(asm.R2, asm.R7, "exit"),

		asm.LoadMem(asm.R8, asm.R6, offV6Proto, asm.Byte),

		// 16 字节源地址
		asm.LoadMem(asm.R2, asm.R6, offV6SrcIP, asm.Word),
		asm.StoreMem(asm.RFP, -20, asm.R2, asm.Word),
		asm.LoadMem(asm.R2, asm.R6, offV6SrcIP+4, asm.Word),
		asm.StoreMem(asm.RFP, -16, asm.R2, asm.Word),
		asm.LoadMem(asm.R2, asm.R6, offV6SrcIP+8, asm.Word),
		asm.StoreMem(asm.RFP, -12, asm.R2, asm.Word),
		asm.LoadMem(asm.R2, asm.R6, offV6SrcIP+12, asm.Word),
		asm.StoreMem(asm.RFP, -8, asm.R2, asm.Word),
		asm.StoreImm(asm.RFP, -1, 6, asm.Byte), // version = 6

		asm.JEq.Imm(asm.R8, ipTCP, "v6tcp"),
		asm.JEq.Imm(asm.R8, ipUDP, "v6udp"),
		asm.Ja.Label("exit"),
	)

	v6t := ethLen + ipv6Len
	insns = append(insns, tcpHandler("v6tcp", v6t+tcpLen, v6t+13, v6t+2)...)
	insns = append(insns, udpHandler("v6udp", v6t+udpLen, v6t+2)...)

	// ========== 退出 ==========
	insns = append(insns,
		asm.Mov.Imm(asm.R0, 0).WithSymbol("exit"),
		asm.Return(),
	)

	return ebpf.NewProgram(&ebpf.ProgramSpec{
		Name:         "scan_detector",
		Type:         ebpf.SchedCLS,
		Instructions: insns,
	})
}

// Supported 检测当前系统是否支持 eBPF
func Supported() bool {
	events, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.RingBuf,
		MaxEntries: ringBufSize,
	})
	if err != nil {
		return false
	}

	ports, err := newPortsMap()
	if err != nil {
		_ = events.Close()
		return false
	}

	prog, err := buildDetector(events, ports)
	if err != nil {
		_ = events.Close()
		_ = ports.Close()
		return false
	}

	_ = prog.Close()
	_ = events.Close()
	_ = ports.Close()
	return true
}

// New 创建 Scanner，加载 eBPF 程序并挂载到指定网卡
func New(ifaces []string, log *slog.Logger) (*Scanner, error) {
	if len(ifaces) == 0 {
		defaultIface := DefaultInterface()
		if defaultIface == "" {
			return nil, errors.New("no available network interface found")
		}
		ifaces = []string{defaultIface}
	}

	events, err := ebpf.NewMap(&ebpf.MapSpec{
		Type:       ebpf.RingBuf,
		MaxEntries: ringBufSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ring buffer: %w", err)
	}

	ports, err := newPortsMap()
	if err != nil {
		_ = events.Close()
		return nil, fmt.Errorf("failed to create ports whitelist map: %w", err)
	}

	prog, err := buildDetector(events, ports)
	if err != nil {
		_ = events.Close()
		_ = ports.Close()
		return nil, fmt.Errorf("failed to load eBPF program: %w", err)
	}

	s := &Scanner{
		prog:      prog,
		events:    events,
		ports:     ports,
		handles:   make(map[string]*ifaceHandle),
		eventsCh:  make(chan Event, 4096),
		stopCh:    make(chan struct{}),
		log:       log,
		lastPorts: make(map[uint16]bool),
	}

	for _, ifaceName := range ifaces {
		if err := s.attach(ifaceName); err != nil {
			_ = s.Close()
			return nil, fmt.Errorf("failed to attach to interface %s: %w", ifaceName, err)
		}
	}

	reader, err := ringbuf.NewReader(events)
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	s.reader = reader

	// 初始同步监听端口白名单
	s.syncPorts()

	go s.readLoop()
	go s.portsLoop()

	return s, nil
}

// Events 返回事件通道
func (s *Scanner) Events() <-chan Event {
	return s.eventsCh
}

// Close 卸载 eBPF 程序并清理资源
func (s *Scanner) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}

	if s.reader != nil {
		_ = s.reader.Close()
	}

	for name, h := range s.handles {
		if err := h.link.Close(); err != nil {
			s.log.Warn("failed to detach eBPF program", slog.String("iface", name), slog.Any("err", err))
		}
		s.log.Info("eBPF scan detector detached", slog.String("iface", name))
	}

	if s.prog != nil {
		_ = s.prog.Close()
	}
	if s.events != nil {
		_ = s.events.Close()
	}
	if s.ports != nil {
		_ = s.ports.Close()
	}

	return nil
}

// attach 挂载 eBPF 程序到指定网卡的 TC ingress
func (s *Scanner) attach(ifaceName string) error {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %w", ifaceName, err)
	}

	l, err := link.AttachTCX(link.TCXOptions{
		Interface: iface.Index,
		Program:   s.prog,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		return fmt.Errorf("failed to attach TCX to %s: %w", ifaceName, err)
	}

	s.handles[ifaceName] = &ifaceHandle{link: l}
	s.log.Info("eBPF scan detector attached", slog.String("iface", ifaceName))

	return nil
}

// readLoop 持续读取 Ring Buffer 事件
func (s *Scanner) readLoop() {
	for {
		record, err := s.reader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			s.log.Warn("failed to read ring buffer", slog.Any("err", err))
			continue
		}

		if len(record.RawSample) < eventSize {
			continue
		}

		select {
		case s.eventsCh <- parseEvent(record.RawSample):
		default:
		}
	}
}

// portsLoop 定时同步监听端口白名单
func (s *Scanner) portsLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.syncPorts()
		}
	}
}

// syncPorts 读取系统当前监听端口并同步到 BPF map
func (s *Scanner) syncPorts() {
	newPorts := readListeningPorts()

	// 删除不再监听的端口
	for port := range s.lastPorts {
		if !newPorts[port] {
			_ = s.ports.Delete(uint32(port))
		}
	}

	// 添加新监听的端口
	for port := range newPorts {
		if !s.lastPorts[port] {
			_ = s.ports.Put(uint32(port), uint8(1))
		}
	}

	s.lastPorts = newPorts
}

// readListeningPorts 从 /proc 读取当前所有监听端口
func readListeningPorts() map[uint16]bool {
	ports := make(map[uint16]bool)
	// TCP LISTEN (state 0A)
	parseProcNet("/proc/net/tcp", "0A", ports)
	parseProcNet("/proc/net/tcp6", "0A", ports)
	// UDP 绑定端口 (state 07 = unconnected)
	parseProcNet("/proc/net/udp", "07", ports)
	parseProcNet("/proc/net/udp6", "07", ports)
	return ports
}

// parseProcNet 解析 /proc/net/{tcp,tcp6,udp,udp6} 提取指定状态的本地端口
func parseProcNet(path, state string, ports map[uint16]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[3] != state {
			continue
		}
		// local_address 格式: "00000000:0050" 或 "00000000000000000000000000000000:0050"
		idx := strings.LastIndex(fields[1], ":")
		if idx < 0 {
			continue
		}
		port, err := strconv.ParseUint(fields[1][idx+1:], 16, 16)
		if err != nil {
			continue
		}
		ports[uint16(port)] = true
	}
}

// parseEvent 解析原始事件数据
// 栈布局：[src_ip(16) | dst_port(2) | protocol(1) | version(1)]
func parseEvent(data []byte) Event {
	version := data[19]

	var ip net.IP
	if version == 6 {
		ip = make(net.IP, 16)
		copy(ip, data[0:16])
	} else {
		ip = net.IPv4(data[0], data[1], data[2], data[3])
	}

	dstPort := binary.LittleEndian.Uint16(data[16:18])
	proto := data[18]

	protoStr := "tcp"
	if proto == ipUDP {
		protoStr = "udp"
	}

	return Event{
		SourceIP:  ip.String(),
		Port:      dstPort,
		Protocol:  protoStr,
		Timestamp: time.Now(),
	}
}
