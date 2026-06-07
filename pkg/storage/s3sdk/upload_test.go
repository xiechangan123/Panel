package s3sdk

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

// TestPut 用 mock S3 服务端验证 Put 的两条路径：
// 小文件走单次 PUT，大文件走流式分片并能正确重组
func TestPut(t *testing.T) {
	var mu sync.Mutex
	parts := map[int][]byte{}
	var simpleBody []byte
	var completeCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			_, _ = io.WriteString(w, `<InitiateMultipartUploadResult><UploadId>uid</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && q.Get("uploadId") != "":
			n, _ := strconv.Atoi(q.Get("partNumber"))
			mu.Lock()
			parts[n] = body
			mu.Unlock()
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodPost && q.Get("uploadId") != "":
			var req struct {
				Parts []completedPart `xml:"Part"`
			}
			_ = xml.Unmarshal(body, &req)
			mu.Lock()
			completeCount = len(req.Parts)
			mu.Unlock()
			_, _ = io.WriteString(w, `<CompleteMultipartUploadResult></CompleteMultipartUploadResult>`)
		case r.Method == http.MethodPut:
			mu.Lock()
			simpleBody = body
			mu.Unlock()
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	c := New(Config{Region: "us-east-1", Bucket: "b", Endpoint: srv.URL, PathStyle: true, Concurrency: 3})

	t.Run("small file uses single PUT", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 1024)
		if err := c.Put("small.txt", bytes.NewReader(data), "text/plain"); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(simpleBody, data) {
			t.Errorf("单次 PUT body 不符: 收到 %d 字节, 期望 %d", len(simpleBody), len(data))
		}
		if len(parts) != 0 {
			t.Errorf("小文件不应触发分片, 却收到 %d 片", len(parts))
		}
	})

	t.Run("large file uses streaming multipart", func(t *testing.T) {
		mu.Lock()
		parts = map[int][]byte{}
		completeCount = 0
		mu.Unlock()

		// 12MB → 5MB + 5MB + 2MB = 3 片
		data := make([]byte, 12*1024*1024)
		for i := range data {
			data[i] = byte(i)
		}
		if err := c.Put("big.bin", bytes.NewReader(data), "application/octet-stream"); err != nil {
			t.Fatal(err)
		}
		if len(parts) != 3 {
			t.Fatalf("期望 3 片, 收到 %d 片", len(parts))
		}
		if completeCount != 3 {
			t.Errorf("complete 收到 %d 片, 期望 3", completeCount)
		}
		// 按片号重组，校验内容完整无错位
		var got []byte
		for i := 1; i <= 3; i++ {
			got = append(got, parts[i]...)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("重组后内容不符: 收到 %d 字节, 期望 %d", len(got), len(data))
		}
	})
}
