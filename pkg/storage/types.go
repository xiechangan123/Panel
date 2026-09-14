package storage

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

type Storage interface {
	// Delete deletes the given file(s).
	Delete(ctx context.Context, file ...string) error
	// Exists determines if a file exists.
	Exists(ctx context.Context, file string) bool
	// LastModified gets the file's last modified time.
	LastModified(ctx context.Context, file string) (time.Time, error)
	// List lists all files (not directories) in the given path.
	List(ctx context.Context, path string) ([]string, error)
	// Put writes the contents of a file.
	Put(ctx context.Context, file string, content io.Reader) error
	// Size gets the file size of a given file.
	Size(ctx context.Context, file string) (int64, error)
}

// Renamer 由支持服务端改名的存储器实现，用于「先传临时名、成功后改名」避免中断留下能以假乱真的半截文件
// S3 不实现：分片上传在取消时会 abort 清理，对象在 complete 之前根本不存在
type Renamer interface {
	Rename(ctx context.Context, src, dst string) error
}

// ctxReader 让裸 io.Copy 响应取消：本地与 sftp 的写入都没有接收 context 的 API
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// newTransport 构造带各阶段超时的传输层
// 不设 http.Client.Timeout：上传耗时随文件大小变化，只能靠分阶段超时兜住对端黑洞
func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ResponseHeaderTimeout: time.Minute,
	}
}
