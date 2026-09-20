package lsapi

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func mustBuild(t *testing.T, params map[string]string) []byte {
	t.Helper()
	buf, err := buildRequest(params)
	must.NoError(t, err)
	return buf
}

// parsed 按 lsphp 侧 parseRequest 的步骤还原出来的请求包内容
type parsed struct {
	total      int
	env        map[string]string
	scriptFile string
	scriptName string
	queryStr   string
	method     string
}

// parseAsServer 复刻 lsapilib.c 的 parseRequest：依次是定长头、特殊环境变量区、
// 环境变量区、8 字节对齐的已知头索引表，最后位置必须正好落在包尾
func parseAsServer(t *testing.T, buf []byte) parsed {
	t.Helper()
	must.True(t, len(buf) > packetHeaderLen)
	check.Equal(t, string(buf[:2]), "LS")
	check.Equal(t, int(buf[2]), typeBeginRequest)
	check.Equal(t, int(buf[3]), int(endianFlag))

	total := int(binary.NativeEndian.Uint32(buf[4:8]))
	must.Equal(t, total, len(buf))

	field := func(i int) int {
		return int(binary.NativeEndian.Uint32(buf[packetHeaderLen+i*4:]))
	}
	httpHeaderLen, cntUnknown := field(0), field(6)
	cntEnv, cntSpecial := field(7), field(8)
	check.Equal(t, httpHeaderLen, 0)
	check.Equal(t, cntUnknown, 0)
	check.Equal(t, cntSpecial, 0)

	pos := packetHeaderLen + 4*9
	readEnv := func(count int) map[string]string {
		out := make(map[string]string, count)
		for range count {
			keyLen := int(binary.BigEndian.Uint16(buf[pos:]))
			valLen := int(binary.BigEndian.Uint16(buf[pos+2:]))
			must.True(t, keyLen > 0 && valLen > 0)
			pos += 4
			key := string(buf[pos : pos+keyLen-1])
			check.Equal(t, int(buf[pos+keyLen-1]), 0)
			pos += keyLen
			value := string(buf[pos : pos+valLen-1])
			check.Equal(t, int(buf[pos+valLen-1]), 0)
			pos += valLen
			out[key] = value
		}
		// 每个区以 4 个 0 收尾
		check.Equal(t, binary.NativeEndian.Uint32(buf[pos:]), uint32(0))
		pos += 4
		return out
	}

	readEnv(cntSpecial)
	env := readEnv(cntEnv)

	pos = (pos + 7) &^ 7
	pos += headerIndexLen
	pos += cntUnknown * 16
	pos += httpHeaderLen
	// parseRequest 最后要求位置正好等于包尾，差一个字节就整包被拒
	must.Equal(t, pos, total)

	str := func(off int) string {
		end := off
		for buf[end] != 0 {
			end++
		}
		return string(buf[off:end])
	}

	return parsed{
		total:      total,
		env:        env,
		scriptFile: str(field(2)),
		scriptName: str(field(3)),
		queryStr:   str(field(4)),
		method:     str(field(5)),
	}
}

func TestBuildRequest(t *testing.T) {
	params := map[string]string{
		"SCRIPT_FILENAME": "/tmp/probe.php",
		"SCRIPT_NAME":     "/probe.php",
		"QUERY_STRING":    "action=reset",
		"REQUEST_METHOD":  "GET",
		"SERVER_PROTOCOL": "HTTP/1.1",
		"HTTP_X_TEST":     "hello",
	}

	got := parseAsServer(t, mustBuild(t, params))

	check.DeepEqual(t, got.env, params)
	check.Equal(t, got.scriptFile, "/tmp/probe.php")
	check.Equal(t, got.scriptName, "/probe.php")
	check.Equal(t, got.queryStr, "action=reset")
	check.Equal(t, got.method, "GET")
	check.Equal(t, got.total%8, headerIndexLen%8)
}

// 四个固定键即使调用方没传也要占位，它们的偏移被请求头引用
func TestBuildRequestMissingFixedKeys(t *testing.T) {
	got := parseAsServer(t, mustBuild(t, map[string]string{"SERVER_NAME": "localhost"}))

	check.Len(t, got.env, 5)
	check.Equal(t, got.scriptFile, "")
	check.Equal(t, got.method, "")
	check.Equal(t, got.env["SERVER_NAME"], "localhost")
}

func TestBuildRequestEmptyParams(t *testing.T) {
	got := parseAsServer(t, mustBuild(t, nil))

	check.Len(t, got.env, 4)
}

// 长度字段只有两字节，超长的值必须被挡住而不是截断成错位的包
func TestBuildRequestOversizedValue(t *testing.T) {
	long := make([]byte, maxEnvLen+1)
	for i := range long {
		long[i] = 'a'
	}

	got := parseAsServer(t, mustBuild(t, map[string]string{
		"HTTP_LONG":   string(long),
		"SERVER_NAME": "localhost",
	}))

	check.Equal(t, got.env["SERVER_NAME"], "localhost")
	_, ok := got.env["HTTP_LONG"]
	check.False(t, ok)
}

// 超过 lsphp 的 readReq 上限时要自己拦下，否则只会收到一个没头没尾的 EOF
func TestBuildRequestTooLarge(t *testing.T) {
	params := make(map[string]string, 3000)
	value := string(make([]byte, 100))
	for i := range 3000 {
		params["HTTP_X_"+strconv.Itoa(i)] = value
	}

	_, err := buildRequest(params)
	check.Error(t, err)
}

// fakeLSPHP 起一个假 lsphp，按给定的包序列应答一次请求
func fakeLSPHP(t *testing.T, packets [][]byte, closeEarly bool) string {
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
		_, _ = io.CopyN(io.Discard, conn, 8) // 请求包头
		for _, p := range packets {
			_, _ = conn.Write(p)
		}
		if !closeEarly {
			time.Sleep(50 * time.Millisecond)
		}
	}()

	return socket
}

func packet(typ byte, payload []byte) []byte {
	buf := make([]byte, packetHeaderLen+len(payload))
	copy(buf, []byte{'L', 'S', typ, endianFlag})
	binary.NativeEndian.PutUint32(buf[4:8], uint32(packetHeaderLen+len(payload))) //nolint:gosec // 测试数据长度可控
	copy(buf[packetHeaderLen:], payload)
	return buf
}

func respHeader(status uint32) []byte {
	payload := make([]byte, 8)
	binary.NativeEndian.PutUint32(payload[4:], status)
	return packet(typeRespHeader, payload)
}

// pidPacket lsphp fork 后借 stderr 通道上报 pid，客户端必须忽略它
func pidPacket() []byte {
	return packet(typeStderrStream, append([]byte("\x00PID"), 1, 2, 3, 4))
}

func request(t *testing.T, socket string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	return Request(ctx, "unix", socket, map[string]string{"SCRIPT_FILENAME": "/tmp/x.php"})
}

func TestRequestBody(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{
		pidPacket(),
		respHeader(200),
		packet(typeRespStream, []byte("hello ")),
		packet(typeRespStream, []byte("world")),
		packet(typeRespEnd, nil),
	}, false)

	body, err := request(t, socket)
	must.NoError(t, err)
	check.Equal(t, string(body), "hello world")
}

// pid 通知包不能被当成错误输出，否则每个请求的 stderr 都非空
func TestRequestIgnoresPIDNotify(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{pidPacket(), respHeader(200), packet(typeRespEnd, nil)}, false)

	body, err := request(t, socket)
	check.NoError(t, err)
	check.Empty(t, body)
}

func TestRequestStderr(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{
		pidPacket(),
		packet(typeStderrStream, []byte("PHP Fatal error: boom")),
		packet(typeRespEnd, nil),
	}, false)

	_, err := request(t, socket)
	must.Error(t, err)
	check.Contains(t, err.Error(), "PHP Fatal error: boom")
	check.NotContains(t, err.Error(), "PID")
}

// 有正文时 stderr 里的 warning 不该让整个请求失败
func TestRequestStderrWithBody(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{
		packet(typeStderrStream, []byte("PHP Warning: deprecated")),
		packet(typeRespStream, []byte("ok")),
		packet(typeRespEnd, nil),
	}, false)

	body, err := request(t, socket)
	must.NoError(t, err)
	check.Equal(t, string(body), "ok")
}

// 没等到 RESP_END 就断开是子进程异常，不能把半截正文当成功
func TestRequestTruncated(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{packet(typeRespStream, []byte("half"))}, true)

	_, err := request(t, socket)
	check.Error(t, err)
}

func TestRequestBadSignature(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{{'X', 'X', 0, 0, 0, 0, 0, 8}}, true)

	_, err := request(t, socket)
	must.Error(t, err)
	check.Contains(t, err.Error(), "bad packet signature")
}

func TestRequestBadLength(t *testing.T) {
	bad := make([]byte, packetHeaderLen)
	copy(bad, []byte{'L', 'S', typeRespStream, endianFlag})
	binary.NativeEndian.PutUint32(bad[4:8], 3) // 比包头还短
	socket := fakeLSPHP(t, [][]byte{bad}, true)

	_, err := request(t, socket)
	must.Error(t, err)
	check.Contains(t, err.Error(), "bad packet length 3")
}

// PHP 报 500 且没有任何输出时，状态码是唯一线索
func TestRequestErrorStatus(t *testing.T) {
	socket := fakeLSPHP(t, [][]byte{respHeader(500), packet(typeRespEnd, nil)}, false)

	_, err := request(t, socket)
	must.Error(t, err)
	check.Contains(t, err.Error(), "status 500")
}

func TestRequestDialError(t *testing.T) {
	_, err := request(t, filepath.Join(t.TempDir(), "missing.sock"))
	check.Error(t, err)
}
