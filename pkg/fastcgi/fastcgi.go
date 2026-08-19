package fastcgi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	typeBeginRequest byte = 1
	typeEndRequest   byte = 3
	typeParams       byte = 4
	typeStdin        byte = 5
	typeStdout       byte = 6
	typeStderr       byte = 7

	roleResponder = 1
)

// Request 向 FastCGI 服务发起一次 responder 请求，返回剥离 CGI 响应头后的 body
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

	const requestID = 1
	// BEGIN_REQUEST：responder 角色，连接不复用
	if err = writeRecord(conn, typeBeginRequest, requestID, []byte{0, roleResponder, 0, 0, 0, 0, 0, 0}); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for name, value := range params {
		encodeNameValue(&buf, name, value)
	}
	if err = writeRecord(conn, typeParams, requestID, buf.Bytes()); err != nil {
		return nil, err
	}
	if err = writeRecord(conn, typeParams, requestID, nil); err != nil {
		return nil, err
	}
	if err = writeRecord(conn, typeStdin, requestID, nil); err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	header := make([]byte, 8)
	for {
		if _, err = io.ReadFull(conn, header); err != nil {
			return nil, err
		}
		contentLength := binary.BigEndian.Uint16(header[4:6])
		content := make([]byte, int(contentLength)+int(header[6]))
		if _, err = io.ReadFull(conn, content); err != nil {
			return nil, err
		}
		switch header[1] {
		case typeStdout:
			stdout.Write(content[:contentLength])
		case typeStderr:
			stderr.Write(content[:contentLength])
		case typeEndRequest:
			if stderr.Len() > 0 {
				return nil, fmt.Errorf("fastcgi stderr: %s", stderr.String())
			}
			// 剥离 CGI 响应头
			if _, body, found := bytes.Cut(stdout.Bytes(), []byte("\r\n\r\n")); found {
				return body, nil
			}
			return stdout.Bytes(), nil
		}
	}
}

// writeRecord 写入一条 FastCGI 记录
func writeRecord(w io.Writer, typ byte, requestID uint16, content []byte) error {
	header := [8]byte{1, typ}
	binary.BigEndian.PutUint16(header[2:4], requestID)
	binary.BigEndian.PutUint16(header[4:6], uint16(len(content))) //nolint:gosec // params 由面板构造，长度远小于 64KB
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if len(content) > 0 {
		if _, err := w.Write(content); err != nil {
			return err
		}
	}
	return nil
}

// encodeNameValue 按 FastCGI name-value 格式编码
func encodeNameValue(buf *bytes.Buffer, name, value string) {
	writeLength := func(n int) {
		if n < 128 {
			buf.WriteByte(byte(n))
			return
		}
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(n)|1<<31) //nolint:gosec // 长度非负
		buf.Write(b[:])
	}
	writeLength(len(name))
	writeLength(len(value))
	buf.WriteString(name)
	buf.WriteString(value)
}
