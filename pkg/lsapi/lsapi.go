package lsapi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"net"
	"runtime"
	"slices"
	"time"
)

const (
	packetHeaderLen = 8

	typeBeginRequest = 1
	typeRespHeader   = 3
	typeRespStream   = 4
	typeRespEnd      = 5
	typeStderrStream = 6

	// 已知头索引表 uint16[25] + int32[25]，中间有 2 字节对齐填充
	headerIndexLen = 25*2 + 2 + 25*4

	maxPacketLen = 16 << 20

	// lsphp 的 readReq 超过这个长度直接拒绝并关连接，自己先拦住才能给出像样的错误
	maxRequestLen = 256 << 10
	// 响应体总量上限，异常的 lsphp 可能一直吐数据
	maxBodyLen = 32 << 20
	// 调用方没给期限时的兜底，否则读循环会一直等
	defaultTimeout = 30 * time.Second

	// 环境变量的键值长度各用两字节表示，含结尾的 0；数量上限取自 lsphp 的 parseEnv
	maxEnvLen   = 65534
	maxEnvCount = 8192
)

// fixedKeys 这四项的值会被请求头的偏移字段引用，必须存在且排在最前
var fixedKeys = []string{"SCRIPT_FILENAME", "SCRIPT_NAME", "QUERY_STRING", "REQUEST_METHOD"}

// endianFlag lsapidef.h 的字节序判断只认 x86，其它架构一律标成大端；两端一致时都不转换，各自按本机顺序读写
var endianFlag = func() byte {
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
		return 0
	}
	return 1
}()

// Request 发起一次请求，返回的响应体不含响应头
func Request(ctx context.Context, network, address string, params map[string]string) ([]byte, error) {
	req, err := buildRequest(params)
	if err != nil {
		return nil, err
	}

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultTimeout)
	}
	_ = conn.SetDeadline(deadline)

	if _, err = conn.Write(req); err != nil {
		return nil, err
	}

	var body, stderr bytes.Buffer
	var status uint32
	header := make([]byte, packetHeaderLen)
	for {
		if _, err = io.ReadFull(conn, header); err != nil {
			// 正常收尾一定有 RESP_END，走到这里多半是子进程挂了，stderr 里有线索
			if stderr.Len() > 0 {
				return nil, fmt.Errorf("lsapi stderr: %s", stderr.String())
			}
			return nil, err
		}
		if header[0] != 'L' || header[1] != 'S' {
			return nil, fmt.Errorf("lsapi: bad packet signature %q", header[:2])
		}

		size := binary.NativeEndian.Uint32(header[4:8])
		length := int(size) - packetHeaderLen
		if length < 0 || length > maxPacketLen {
			return nil, fmt.Errorf("lsapi: bad packet length %d", size)
		}
		payload := make([]byte, length)
		if _, err = io.ReadFull(conn, payload); err != nil {
			return nil, err
		}

		switch header[2] {
		case typeRespStream:
			if body.Len()+len(payload) > maxBodyLen {
				return nil, fmt.Errorf("lsapi: response exceeds %d bytes", maxBodyLen)
			}
			body.Write(payload)
		case typeStderrStream:
			// prefork 出来的子进程借 stderr 通道上报自己的 pid，不是错误输出
			if bytes.HasPrefix(payload, []byte("\x00PID")) {
				continue
			}
			stderr.Write(payload)
		case typeRespHeader:
			if len(payload) >= 8 {
				status = binary.NativeEndian.Uint32(payload[4:8])
			}
		case typeRespEnd:
			if body.Len() > 0 {
				return body.Bytes(), nil
			}
			if stderr.Len() > 0 {
				return nil, fmt.Errorf("lsapi stderr: %s", stderr.String())
			}
			if status >= 400 {
				return nil, fmt.Errorf("lsapi: php returned status %d", status)
			}
			return nil, nil
		}
	}
}

// buildRequest 定长头 + 环境变量区 + 8 字节对齐后的已知头索引表。
// 头里的四个偏移指向环境变量区中的值，所以这四项要排在最前
func buildRequest(params map[string]string) ([]byte, error) {
	keys := make([]string, len(fixedKeys), len(params)+len(fixedKeys))
	copy(keys, fixedKeys)
	for _, name := range slices.Sorted(maps.Keys(params)) {
		// 超长的可选项直接丢掉，写进去只会让包错位
		if slices.Contains(fixedKeys, name) || len(name) > maxEnvLen || len(params[name]) > maxEnvLen {
			continue
		}
		keys = append(keys, name)
	}
	if len(keys) > maxEnvCount {
		keys = keys[:maxEnvCount]
	}

	reqHeaderLen := packetHeaderLen + 4*9
	envLen := 4 + 4 // 两个环境变量区各有 4 字节结束符，特殊变量区只有结束符
	for _, name := range keys {
		value := params[name]
		if len(value) > maxEnvLen {
			value = ""
		}
		envLen += 4 + len(name) + 1 + len(value) + 1
	}

	total := reqHeaderLen + envLen
	pad := (8 - total%8) % 8
	total += pad + headerIndexLen

	if total > maxRequestLen {
		return nil, fmt.Errorf("lsapi: request packet too large (%d bytes)", total)
	}

	buf := make([]byte, total)
	copy(buf, []byte{'L', 'S', typeBeginRequest, endianFlag})
	binary.NativeEndian.PutUint32(buf[4:8], uint32(total))

	pos := reqHeaderLen + 4 // 跳过空的特殊环境变量区

	offsets := make([]uint32, 4)
	for i, name := range keys {
		value := params[name]
		// 固定项不能丢，超长时退化成空值
		if len(value) > maxEnvLen {
			value = ""
		}
		binary.BigEndian.PutUint16(buf[pos:], uint16(len(name)+1))    //nolint:gosec
		binary.BigEndian.PutUint16(buf[pos+2:], uint16(len(value)+1)) //nolint:gosec
		pos += 4
		pos += copy(buf[pos:], name)
		pos++
		if i < 4 {
			offsets[i] = uint32(pos) //nolint:gosec
		}
		pos += copy(buf[pos:], value)
		pos++
	}
	// 环境变量区结束符与对齐填充都是零值，不用再写

	// 请求头余下的字段：正文长度、请求体长度、四个偏移、三个计数
	fields := []uint32{
		0, 0,
		offsets[0], offsets[1], offsets[2], offsets[3],
		0, uint32(len(keys)), 0, //nolint:gosec
	}
	for i, v := range fields {
		binary.NativeEndian.PutUint32(buf[packetHeaderLen+i*4:], v)
	}

	return buf, nil
}
