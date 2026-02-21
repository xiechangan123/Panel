package ipdb

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

const testDBPath = "../geoip/ipipfree.ipdb"

type ReaderSuite struct {
	suite.Suite
	r *Reader
}

func (s *ReaderSuite) SetupSuite() {
	r, err := Open(testDBPath)
	s.Require().NoError(err)
	s.Require().NotNil(r)
	s.r = r
}

func (s *ReaderSuite) TearDownSuite() {
	s.Require().NoError(s.r.Close())
}

func TestReaderSuite(t *testing.T) {
	suite.Run(t, new(ReaderSuite))
}

// ========== Open ==========

func (s *ReaderSuite) TestOpen_InvalidPath() {
	_, err := Open("/nonexistent/path.ipdb")
	s.Error(err)
}

// ========== Fields ==========

func (s *ReaderSuite) TestFields() {
	fields := s.r.Fields()
	s.NotEmpty(fields)
	s.T().Logf("fields: %v", fields)
}

// ========== Find ==========

func (s *ReaderSuite) TestFind_IPv4() {
	result, err := s.r.Find("114.114.114.114", "CN")
	s.NoError(err)
	s.NotEmpty(result)
	s.T().Logf("114.114.114.114 -> %v", result)
}

func (s *ReaderSuite) TestFind_IPv4_Foreign() {
	result, err := s.r.Find("8.8.8.8", "CN")
	s.NoError(err)
	s.NotEmpty(result)
	s.T().Logf("8.8.8.8 -> %v", result)
}

func (s *ReaderSuite) TestFind_IPv6() {
	// IPv6 查找，可能不支持但不应 panic
	_, _ = s.r.Find("2001:4860:4860::8888", "CN")
}

func (s *ReaderSuite) TestFind_InvalidIP() {
	_, err := s.r.Find("not-an-ip", "CN")
	s.ErrorIs(err, ErrInvalidIP)
}

func (s *ReaderSuite) TestFind_InvalidLanguage() {
	_, err := s.r.Find("8.8.8.8", "INVALID")
	s.ErrorIs(err, ErrNoLanguage)
}

// ========== Reload ==========

func (s *ReaderSuite) TestReload() {
	err := s.r.Reload(testDBPath)
	s.NoError(err)

	// reload 后查询仍正常
	result, err := s.r.Find("114.114.114.114", "CN")
	s.NoError(err)
	s.NotEmpty(result)
}

func (s *ReaderSuite) TestReload_InvalidPath() {
	err := s.r.Reload("/nonexistent/path.ipdb")
	s.Error(err)

	// 失败后原数据仍可用
	result, err := s.r.Find("114.114.114.114", "CN")
	s.NoError(err)
	s.NotEmpty(result)
}

// ========== Close ==========

func (s *ReaderSuite) TestClose_UseAfterClose() {
	r, err := Open(testDBPath)
	s.Require().NoError(err)

	s.NoError(r.Close())

	_, err = r.Find("8.8.8.8", "CN")
	s.ErrorIs(err, ErrClosed)
}

func (s *ReaderSuite) TestClose_DoubleClose() {
	r, err := Open(testDBPath)
	s.Require().NoError(err)

	s.NoError(r.Close())
	s.NoError(r.Close())
}

func (s *ReaderSuite) TestClose_NilReceiver() {
	var r *Reader
	s.NoError(r.Close())
}

// ========== Benchmark ==========

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
