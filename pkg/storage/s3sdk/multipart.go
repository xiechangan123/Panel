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
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	// MinPartSize 是 S3 规定的最小分片大小（最后一片除外）。
	MinPartSize = 5 * 1024 * 1024
	// MaxParts 是单次分片上传的最大分片数。
	MaxParts = 10000
	// DefaultPartSize 是默认分片大小。
	DefaultPartSize = 5 * 1024 * 1024

	defaultMaxRetries = 3
)

// ProgressFunc 在上传过程中被回调以报告进度；并行上传时可能被并发调用。
type ProgressFunc func(ProgressInfo)

// ProgressInfo 描述一次分片上传的进度。
type ProgressInfo struct {
	TotalBytes     int64
	UploadedBytes  int64
	CurrentPart    int
	TotalParts     int
	BytesPerSecond int64
}

// MultipartUploadInput 是 FileUploadMultipart 的入参。
type MultipartUploadInput struct {
	Bucket      string    // bucket 名称
	ObjectKey   string    // 对象 key
	Body        io.Reader // 要上传的数据
	ContentType string    // 可选：内容类型
	PartSize    int64     // 可选：分片大小，默认 5MB，最小 5MB
	MaxRetries  int       // 可选：单分片重试次数，默认 3
	Concurrency int       // 可选：并发分片数，默认 1（顺序上传）
	OnProgress  ProgressFunc
}

// MultipartUploadOutput 是 FileUploadMultipart 的返回。
type MultipartUploadOutput struct {
	Location string
	Bucket   string
	Key      string
	ETag     string
	UploadID string
}

// FileUploadMultipart 以分片方式上传整个对象，失败时自动中止并清理已上传的分片。
func (c *S3) FileUploadMultipart(in MultipartUploadInput) (MultipartUploadOutput, error) {
	if in.Body == nil {
		return MultipartUploadOutput{}, errors.New("s3: body is required")
	}

	partSize := in.PartSize
	if partSize == 0 {
		partSize = DefaultPartSize
	}
	if partSize < MinPartSize {
		return MultipartUploadOutput{}, fmt.Errorf("s3: part size must be at least %d bytes", MinPartSize)
	}
	maxRetries := in.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	concurrency := in.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	data, err := io.ReadAll(in.Body)
	if err != nil {
		return MultipartUploadOutput{}, err
	}

	totalParts := int((int64(len(data)) + partSize - 1) / partSize)
	if totalParts > MaxParts {
		return MultipartUploadOutput{}, fmt.Errorf("s3: file too large: needs %d parts, max is %d", totalParts, MaxParts)
	}

	uploadID, err := c.initiateMultipartUpload(in)
	if err != nil {
		return MultipartUploadOutput{}, err
	}

	parts, err := c.uploadParts(data, partSize, totalParts, concurrency, maxRetries, in, uploadID)
	if err != nil {
		_ = c.abortMultipartUpload(in.Bucket, in.ObjectKey, uploadID)
		return MultipartUploadOutput{}, err
	}

	out, err := c.completeMultipartUpload(in.Bucket, in.ObjectKey, uploadID, parts)
	if err != nil {
		_ = c.abortMultipartUpload(in.Bucket, in.ObjectKey, uploadID)
		return MultipartUploadOutput{}, err
	}
	return out, nil
}

type completedPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

// uploadParts 并发上传所有分片，结果按分片序号原位写入因而天然有序。
// concurrency 为 1 时即顺序上传。
func (c *S3) uploadParts(data []byte, partSize int64, totalParts, concurrency, maxRetries int, in MultipartUploadInput, uploadID string) ([]completedPart, error) {
	totalSize := int64(len(data))
	parts := make([]completedPart, totalParts)

	var uploaded, completed atomic.Int64
	start := time.Now()

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(concurrency)

	for i := range totalParts {
		partNum := i + 1
		lo := int64(i) * partSize
		hi := min(lo+partSize, totalSize)
		chunk := data[lo:hi]

		g.Go(func() error {
			etag, err := c.uploadPart(ctx, uploadID, in.Bucket, in.ObjectKey, partNum, chunk, maxRetries)
			if err != nil {
				return err
			}
			parts[i] = completedPart{PartNumber: partNum, ETag: etag}

			if in.OnProgress != nil {
				u := uploaded.Add(int64(len(chunk)))
				n := completed.Add(1)
				var bps int64
				if elapsed := time.Since(start).Seconds(); elapsed > 0 {
					bps = int64(float64(u) / elapsed)
				}
				in.OnProgress(ProgressInfo{
					TotalBytes:     totalSize,
					UploadedBytes:  u,
					CurrentPart:    int(n),
					TotalParts:     totalParts,
					BytesPerSecond: bps,
				})
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return parts, nil
}

// uploadPart 上传单个分片，带指数退避重试。
func (c *S3) uploadPart(ctx context.Context, uploadID, bucket, objectKey string, partNum int, chunk []byte, maxRetries int) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := min(time.Duration(1<<(attempt-1))*100*time.Millisecond, 5*time.Second)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(wait):
			}
		}

		etag, err := c.putPart(uploadID, bucket, objectKey, partNum, chunk)
		if err == nil {
			return etag, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("s3: upload part %d failed after %d retries: %w", partNum, maxRetries, lastErr)
}

func (c *S3) putPart(uploadID, bucket, objectKey string, partNum int, chunk []byte) (string, error) {
	query := url.Values{
		"partNumber": {strconv.Itoa(partNum)},
		"uploadId":   {uploadID},
	}
	req, err := http.NewRequest(http.MethodPut, c.buildURL(bucket, objectKey)+"?"+query.Encode(), bytes.NewReader(chunk))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(chunk))
	req.Header.Set("x-amz-content-sha256", sha256Hex(chunk))

	_, res, err := c.do(req, http.StatusOK)
	if err != nil {
		return "", err
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		return "", errors.New("s3: missing ETag in upload part response")
	}
	return etag, nil
}

// isRetryable 判断错误是否值得重试：5xx 服务端错误或网络层错误。
func isRetryable(err error) bool {
	if ae, ok := errors.AsType[*apiError](err); ok {
		return ae.status >= 500
	}
	var ne net.Error
	return errors.As(err, &ne)
}

func (c *S3) initiateMultipartUpload(in MultipartUploadInput) (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.buildURL(in.Bucket, in.ObjectKey)+"?uploads", nil)
	if err != nil {
		return "", err
	}
	if in.ContentType != "" {
		req.Header.Set("Content-Type", in.ContentType)
	}

	body, _, err := c.do(req, http.StatusOK)
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

func (c *S3) completeMultipartUpload(bucket, objectKey, uploadID string, parts []completedPart) (MultipartUploadOutput, error) {
	payload := struct {
		XMLName xml.Name        `xml:"CompleteMultipartUpload"`
		XMLNS   string          `xml:"xmlns,attr"`
		Parts   []completedPart `xml:"Part"`
	}{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Parts: parts}

	body, err := xml.Marshal(payload)
	if err != nil {
		return MultipartUploadOutput{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.buildURL(bucket, objectKey)+"?uploadId="+url.QueryEscape(uploadID), bytes.NewReader(body))
	if err != nil {
		return MultipartUploadOutput{}, err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("x-amz-content-sha256", sha256Hex(body))
	md5sum := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5sum[:]))

	respBody, _, err := c.do(req, http.StatusOK)
	if err != nil {
		return MultipartUploadOutput{}, err
	}

	var result struct {
		Location string `xml:"Location"`
		Bucket   string `xml:"Bucket"`
		Key      string `xml:"Key"`
		ETag     string `xml:"ETag"`
	}
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return MultipartUploadOutput{}, fmt.Errorf("s3: parse complete response: %w", err)
	}
	return MultipartUploadOutput{
		Location: result.Location,
		Bucket:   result.Bucket,
		Key:      result.Key,
		ETag:     result.ETag,
		UploadID: uploadID,
	}, nil
}

func (c *S3) abortMultipartUpload(bucket, objectKey, uploadID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.buildURL(bucket, objectKey)+"?uploadId="+url.QueryEscape(uploadID), nil)
	if err != nil {
		return err
	}
	_, _, err = c.do(req, http.StatusNoContent)
	return err
}
