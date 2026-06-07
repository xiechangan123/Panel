package s3sdk

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	// MinPartSize 是 S3 规定的最小分片大小（最后一片除外）
	MinPartSize = 5 * 1024 * 1024
	// MaxParts 是单次分片上传的最大分片数
	MaxParts = 10000
	// DefaultPartSize 是默认分片大小
	DefaultPartSize = 5 * 1024 * 1024

	defaultMaxRetries = 3
)

// Put 上传一个对象：内容不足一个分片时走单次 PUT，否则流式分片上传
func (c *S3) Put(key string, body io.Reader, contentType string) error {
	// 用动态 buffer 读取首块，避免小对象也分配整个 partSize
	var first bytes.Buffer
	n, err := io.CopyN(&first, body, c.partSize)
	if err != nil && err != io.EOF {
		return err
	}
	if n < c.partSize {
		// 全部内容不足一个分片 → 单次 PUT（含空对象）
		return c.putObject(key, first.Bytes(), contentType)
	}
	// 已读满一个分片，可能还有更多 → 流式分片上传
	return c.uploadMultipart(key, first.Bytes(), body, contentType)
}

func (c *S3) putObject(key string, data []byte, contentType string) error {
	req, err := http.NewRequest(http.MethodPut, c.objectURL(key), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("x-amz-content-sha256", sha256Hex(data))

	_, _, err = c.do(context.Background(), req, http.StatusOK)
	return err
}

type completedPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

func (c *S3) uploadMultipart(key string, first []byte, body io.Reader, contentType string) error {
	uploadID, err := c.initiate(key, contentType)
	if err != nil {
		return err
	}

	parts, err := c.uploadParts(key, uploadID, first, body)
	if err != nil {
		_ = c.abort(key, uploadID)
		return err
	}
	if err := c.complete(key, uploadID, parts); err != nil {
		_ = c.abort(key, uploadID)
		return err
	}
	return nil
}

// uploadParts 顺序读取、并发上传分片。errgroup 的并发上限会对读取形成背压，
// 因此常驻内存约为 concurrency × partSize，而非整个文件大小
func (c *S3) uploadParts(key, uploadID string, first []byte, body io.Reader) ([]completedPart, error) {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(c.concurrency)

	var mu sync.Mutex
	var parts []completedPart

	submit := func(partNum int, chunk []byte) {
		g.Go(func() error {
			etag, err := c.putPart(ctx, key, uploadID, partNum, chunk)
			if err != nil {
				return err
			}
			mu.Lock()
			parts = append(parts, completedPart{PartNumber: partNum, ETag: etag})
			mu.Unlock()
			return nil
		})
	}

	submit(1, first)

	// 某分片失败时 errgroup 会取消 ctx，循环随即停止继续读取大文件
	for partNum := 2; ctx.Err() == nil; partNum++ {
		buf := make([]byte, c.partSize)
		n, err := io.ReadFull(body, buf)
		if n > 0 {
			submit(partNum, buf[:n])
		}
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				_ = g.Wait()
				return nil, err
			}
			break
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 并发完成顺序不定，complete 要求按分片号升序
	slices.SortFunc(parts, func(a, b completedPart) int { return a.PartNumber - b.PartNumber })
	return parts, nil
}

func (c *S3) putPart(ctx context.Context, key, uploadID string, partNum int, chunk []byte) (string, error) {
	query := url.Values{
		"partNumber": {strconv.Itoa(partNum)},
		"uploadId":   {uploadID},
	}
	req, err := http.NewRequest(http.MethodPut, c.objectURL(key)+"?"+query.Encode(), bytes.NewReader(chunk))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(chunk))
	req.Header.Set("x-amz-content-sha256", sha256Hex(chunk))

	_, res, err := c.do(ctx, req, http.StatusOK)
	if err != nil {
		return "", err
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		return "", errors.New("s3: missing ETag in upload part response")
	}
	return etag, nil
}

// isRetryable 判断错误是否值得重试：5xx 服务端错误或网络层错误
func isRetryable(err error) bool {
	if ae, ok := errors.AsType[*apiError](err); ok {
		return ae.status >= 500
	}
	var ne net.Error
	return errors.As(err, &ne)
}

func (c *S3) initiate(key, contentType string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.objectURL(key)+"?uploads", nil)
	if err != nil {
		return "", err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	body, _, err := c.do(context.Background(), req, http.StatusOK)
	if err != nil {
		return "", err
	}

	var result struct {
		UploadID string `xml:"UploadId"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("s3: parse initiate response: %w", err)
	}
	if result.UploadID == "" {
		return "", errors.New("s3: empty upload id in initiate response")
	}
	return result.UploadID, nil
}

func (c *S3) complete(key, uploadID string, parts []completedPart) error {
	payload := struct {
		XMLName xml.Name        `xml:"CompleteMultipartUpload"`
		XMLNS   string          `xml:"xmlns,attr"`
		Parts   []completedPart `xml:"Part"`
	}{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Parts: parts}

	body, err := xml.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.objectURL(key)+"?uploadId="+url.QueryEscape(uploadID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("x-amz-content-sha256", sha256Hex(body))
	sum := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))

	_, _, err = c.do(context.Background(), req, http.StatusOK)
	return err
}

func (c *S3) abort(key, uploadID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.objectURL(key)+"?uploadId="+url.QueryEscape(uploadID), nil)
	if err != nil {
		return err
	}
	_, _, err = c.do(context.Background(), req, http.StatusNoContent)
	return err
}
