package lsapi

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
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
)

// endianFlag lsapidef.h 的字节序判断只认 x86，其它架构一律标成大端；两端一致时都不转换，各自按本机顺序读写
var endianFlag = func() byte {
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
		return 0
	}
	return 1
}()

// Request 发起一次请求，返回的响应体不含响应头
func Request(ctx context.Context, network, address string, params map[string]string) ([]byte, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if _, err = conn.Write(buildRequest(params)); err != nil {
		return nil, err
	}

	var body, stderr bytes.Buffer
	header := make([]byte, packetHeaderLen)
	for {
		if _, err = io.ReadFull(conn, header); err != nil {
			// lsphp 处理完可能直接关连接，已经收到的就是完整响应
			if errors.Is(err, io.EOF) && body.Len() > 0 {
				return body.Bytes(), nil
			}
			return nil, err
		}
		if header[0] != 'L' || header[1] != 'S' {
			return nil, fmt.Errorf("lsapi: bad packet signature %q", header[:2])
		}

		length := int(binary.NativeEndian.Uint32(header[4:8])) - packetHeaderLen
		if length < 0 || length > maxPacketLen {
			return nil, fmt.Errorf("lsapi: bad packet length %d", length)
		}
		payload := make([]byte, length)
		if _, err = io.ReadFull(conn, payload); err != nil {
			return nil, err
		}

		switch header[2] {
		case typeRespStream:
			body.Write(payload)
		case typeStderrStream:
			stderr.Write(payload)
		case typeRespEnd:
			if stderr.Len() > 0 && body.Len() == 0 {
				return nil, fmt.Errorf("lsapi stderr: %s", stderr.String())
			}
			return body.Bytes(), nil
		}
	}
}

// buildRequest 定长头 + 环境变量区 + 8 字节对齐后的已知头索引表。
// 头里的四个偏移指向环境变量区中的值，所以这四项要排在最前
func buildRequest(params map[string]string) []byte {
	keys := []string{"SCRIPT_FILENAME", "SCRIPT_NAME", "QUERY_STRING", "REQUEST_METHOD"}
	for name := range params {
		switch name {
		case "SCRIPT_FILENAME", "SCRIPT_NAME", "QUERY_STRING", "REQUEST_METHOD":
		default:
			keys = append(keys, name)
		}
	}

	reqHeaderLen := packetHeaderLen + 4*9
	envLen := 4 + 4 // 两个环境变量区各有 4 字节结束符，特殊变量区只有结束符
	for _, name := range keys {
		envLen += 4 + len(name) + 1 + len(params[name]) + 1
	}

	total := reqHeaderLen + envLen
	pad := (8 - total%8) % 8
	total += pad + headerIndexLen

	buf := make([]byte, total)
	copy(buf, []byte{'L', 'S', typeBeginRequest, endianFlag})
	binary.NativeEndian.PutUint32(buf[4:8], uint32(total))

	pos := reqHeaderLen + 4 // 跳过空的特殊环境变量区

	offsets := make([]uint32, 4)
	for i, name := range keys {
		value := params[name]
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

	return buf
}
