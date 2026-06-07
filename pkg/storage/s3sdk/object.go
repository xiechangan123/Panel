package s3sdk

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ObjectInfo 是对象元数据，来自 HEAD 响应并已解析为强类型
type ObjectInfo struct {
	ContentType  string
	Size         int64
	ETag         string
	LastModified time.Time
	Metadata     map[string]string // x-amz-meta-*（已去除前缀）
}

// Stat 通过 HEAD 获取对象元数据；对象不存在时返回的错误可用 IsNotFound 判断
func (c *S3) Stat(key string) (ObjectInfo, error) {
	req, err := http.NewRequest(http.MethodHead, c.objectURL(key), nil)
	if err != nil {
		return ObjectInfo{}, err
	}

	_, res, err := c.do(context.Background(), req, http.StatusOK)
	if err != nil {
		return ObjectInfo{}, err
	}

	info := ObjectInfo{
		ContentType: res.Header.Get("Content-Type"),
		ETag:        res.Header.Get("ETag"),
	}
	if v := res.Header.Get("Content-Length"); v != "" {
		info.Size, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := res.Header.Get("Last-Modified"); v != "" {
		info.LastModified, _ = http.ParseTime(v)
	}
	for k, vs := range res.Header {
		if lk := strings.ToLower(k); strings.HasPrefix(lk, amzMetaPrefix) {
			if info.Metadata == nil {
				info.Metadata = make(map[string]string)
			}
			info.Metadata[strings.TrimPrefix(lk, amzMetaPrefix)] = vs[0]
		}
	}
	return info, nil
}

// Delete 删除一个或多个对象，自动按每批 1000 个分批请求
func (c *S3) Delete(keys ...string) error {
	for batch := range slices.Chunk(keys, 1000) {
		if err := c.deleteBatch(batch); err != nil {
			return err
		}
	}
	return nil
}

func (c *S3) deleteBatch(keys []string) error {
	type object struct {
		Key string `xml:"Key"`
	}
	payload := struct {
		XMLName xml.Name `xml:"Delete"`
		XMLNS   string   `xml:"xmlns,attr"`
		Quiet   bool     `xml:"Quiet"`
		Objects []object `xml:"Object"`
	}{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Quiet: true}
	for _, key := range keys {
		payload.Objects = append(payload.Objects, object{Key: key})
	}

	body, err := xml.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.base+"?delete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("x-amz-content-sha256", sha256Hex(body))
	// S3 对 DeleteObjects 强制要求 Content-MD5
	sum := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(sum[:]))

	respBody, _, err := c.do(context.Background(), req, http.StatusOK)
	if err != nil {
		return err
	}

	// Quiet 模式下响应只含失败项
	var result struct {
		Errors []struct {
			Key     string `xml:"Key"`
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	}
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("s3: parse delete response: %w", err)
	}
	if len(result.Errors) > 0 {
		e := result.Errors[0]
		return fmt.Errorf("s3: delete failed for %q: %s: %s (%d errors total)", e.Key, e.Code, e.Message, len(result.Errors))
	}
	return nil
}
