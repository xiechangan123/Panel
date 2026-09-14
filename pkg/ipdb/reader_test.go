package ipdb

import (
	"io/fs"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

const testDBPath = "../geoip/ipipfree.ipdb"

func newReader(t *testing.T) *Reader {
	t.Helper()
	r, err := Open(testDBPath)
	must.NoError(t, err)
	t.Cleanup(func() { check.NoError(t, r.Close()) })
	return r
}

// mustFind 查询并校验返回值个数与字段数一致——库内容会变，只有这个数量关系恒定
func mustFind(t *testing.T, r *Reader, ip string) []string {
	t.Helper()
	result, err := r.Find(ip, "CN")
	must.NoError(t, err)
	must.Len(t, result, len(r.Fields()))
	return result
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path.ipdb")
	check.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFields(t *testing.T) {
	fields := newReader(t).Fields()
	check.NotEmpty(t, fields)
	t.Logf("fields: %v", fields)
}

func TestFind_IPv4(t *testing.T) {
	r := newReader(t)
	for _, ip := range []string{"114.114.114.114", "8.8.8.8"} {
		t.Run(ip, func(t *testing.T) {
			result := mustFind(t, r, ip)
			check.NotEmpty(t, result[0], check.Msgf("%s -> %v", ip, result))
		})
	}
}

func TestFind_IPv6(t *testing.T) {
	r := newReader(t)
	// 库可能不含 IPv6 数据，允许返回错误，但不能 panic
	check.NotPanics(t, func() { _, _ = r.Find("2001:4860:4860::8888", "CN") })
}

func TestFind_InvalidIP(t *testing.T) {
	_, err := newReader(t).Find("not-an-ip", "CN")
	check.ErrorIs(t, err, ErrInvalidIP)
}

func TestFind_InvalidLanguage(t *testing.T) {
	_, err := newReader(t).Find("8.8.8.8", "INVALID")
	check.ErrorIs(t, err, ErrNoLanguage)
}

func TestReload(t *testing.T) {
	r := newReader(t)
	must.NoError(t, r.Reload(testDBPath))

	check.NotEmpty(t, mustFind(t, r, "114.114.114.114")[0])
}

func TestReload_InvalidPath(t *testing.T) {
	r := newReader(t)
	check.ErrorIs(t, r.Reload("/nonexistent/path.ipdb"), fs.ErrNotExist)

	// 失败后原数据仍可用
	check.NotEmpty(t, mustFind(t, r, "114.114.114.114")[0])
}

func TestClose_UseAfterClose(t *testing.T) {
	r, err := Open(testDBPath)
	must.NoError(t, err)

	check.NoError(t, r.Close())

	_, err = r.Find("8.8.8.8", "CN")
	check.ErrorIs(t, err, ErrClosed)
}

func TestClose_DoubleClose(t *testing.T) {
	r, err := Open(testDBPath)
	must.NoError(t, err)

	check.NoError(t, r.Close())
	check.NoError(t, r.Close())
}

func TestClose_NilReceiver(t *testing.T) {
	var r *Reader
	check.NoError(t, r.Close())
}

func BenchmarkFind(b *testing.B) {
	r, err := Open(testDBPath)
	if err != nil {
		b.Fatal(err)
	}
	defer func(r *Reader) { _ = r.Close() }(r)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = r.Find("114.114.114.114", "CN")
	}
}
