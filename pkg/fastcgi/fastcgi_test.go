package fastcgi

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

// record 一条 FastCGI 记录：8 字节头加内容，不带填充
func record(typ byte, content []byte) []byte {
	buf := make([]byte, 8+len(content))
	buf[0], buf[1] = 1, typ
	binary.BigEndian.PutUint16(buf[2:4], 1)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(content))) //nolint:gosec // 测试数据长度可控
	copy(buf[8:], content)
	return buf
}

func stdout(body string) []byte {
	return record(typeStdout, []byte("Content-Type: text/html\r\n\r\n"+body))
}

// fakeFPM 起一个假 php-fpm，收完请求后按给定记录应答
func fakeFPM(t *testing.T, records ...[]byte) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "s.sock")
	ln, err := net.Listen("unix", socket)
	must.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// 请求侧的内容不影响本用例，读掉即可
		go func() { _, _ = io.Copy(io.Discard, conn) }()
		for _, r := range records {
			_, _ = conn.Write(r)
		}
		time.Sleep(50 * time.Millisecond)
	}()

	return socket
}

func request(t *testing.T, socket string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	return Request(ctx, "unix", socket, map[string]string{"SCRIPT_FILENAME": "/tmp/x.php"})
}

func TestRequestBody(t *testing.T) {
	socket := fakeFPM(t, stdout("hello"), record(typeEndRequest, make([]byte, 8)))

	body, err := request(t, socket)
	must.NoError(t, err)
	check.Equal(t, string(body), "hello")
}

func TestRequestStderr(t *testing.T) {
	socket := fakeFPM(t,
		record(typeStderr, []byte("PHP Fatal error: boom")),
		record(typeEndRequest, make([]byte, 8)),
	)

	_, err := request(t, socket)
	must.Error(t, err)
	check.Contains(t, err.Error(), "PHP Fatal error: boom")
}

func TestRequestStderrWithBody(t *testing.T) {
	socket := fakeFPM(t,
		record(typeStderr, []byte("PHP Warning: deprecated")),
		stdout("ok"),
		record(typeEndRequest, make([]byte, 8)),
	)

	body, err := request(t, socket)
	must.NoError(t, err)
	check.Equal(t, string(body), "ok")
}

func TestRequestDialError(t *testing.T) {
	_, err := request(t, filepath.Join(t.TempDir(), "missing.sock"))
	check.Error(t, err)
}
