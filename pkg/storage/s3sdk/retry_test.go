package s3sdk

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// TestDoRetry 验证 do 对 5xx 自动重试，且重试时能用 GetBody 重放请求体
func TestDoRetry(t *testing.T) {
	var attempts atomic.Int32
	var mu sync.Mutex
	var lastBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable) // 首次返回 503，触发重试
			return
		}
		mu.Lock()
		lastBody = string(body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{Region: "us-east-1", Bucket: "b", Endpoint: srv.URL, PathStyle: true})
	if err := c.putObject("k", []byte("hello"), "text/plain"); err != nil {
		t.Fatal(err)
	}

	if got := attempts.Load(); got != 2 {
		t.Errorf("期望重试后共请求 2 次, 实际 %d", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if lastBody != "hello" {
		t.Errorf("重试时请求体未正确重放: 收到 %q", lastBody)
	}
}
