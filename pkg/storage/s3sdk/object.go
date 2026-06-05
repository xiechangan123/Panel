package s3sdk

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DetailsInput 是 FileDetails 的入参。
type DetailsInput struct {
	Bucket    string
	ObjectKey string
	VersionId string // 可选：指定对象版本
}

// DetailsResponse 是 FileDetails 的返回，对应对象 HEAD 响应的常用元数据。
type DetailsResponse struct {
	ContentType   string
	ContentLength string
	ETag          string
	LastModified  string
	AmzMeta       map[string]string // x-amz-meta-* 自定义元数据（已去除前缀）
}

// FileDetails 通过 HEAD 请求获取对象的元数据。
func (c *S3) FileDetails(in DetailsInput) (DetailsResponse, error) {
	u := c.buildURL(in.Bucket, in.ObjectKey)
	if in.VersionId != "" {
		u += "?versionId=" + url.QueryEscape(in.VersionId)
	}

	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return DetailsResponse{}, err
	}

	_, res, err := c.do(req, http.StatusOK)
	if err != nil {
		return DetailsResponse{}, err
	}

	out := DetailsResponse{
		ContentType:   res.Header.Get("Content-Type"),
		ContentLength: res.Header.Get("Content-Length"),
		ETag:          res.Header.Get("ETag"),
		LastModified:  res.Header.Get("Last-Modified"),
	}
	for k, v := range res.Header {
		if lk := strings.ToLower(k); strings.HasPrefix(lk, AMZMetaPrefix) {
			if out.AmzMeta == nil {
				out.AmzMeta = make(map[string]string)
			}
			out.AmzMeta[strings.TrimPrefix(lk, AMZMetaPrefix)] = v[0]
		}
	}
	return out, nil
}

// DeleteObjectsInput 是 DeleteObjects 的入参。
type DeleteObjectsInput struct {
	Bucket  string   // bucket 名称
	Objects []string // 要删除的对象 key（单次最多 1000 个）
	Quiet   bool     // 静默模式：仅返回失败项
}

// DeleteObjectsOutput 是 DeleteObjects 的返回。
type DeleteObjectsOutput struct {
	Deleted []DeletedObject
	Errors  []DeleteError
}

// DeletedObject 表示一个被成功删除的对象。
type DeletedObject struct {
	Key string `xml:"Key"`
}

// DeleteError 表示一个对象的删除错误。
type DeleteError struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// DeleteObjects 在单次请求中批量删除对象（最多 1000 个），同时返回成功项与失败项。
func (c *S3) DeleteObjects(in DeleteObjectsInput) (DeleteObjectsOutput, error) {
	if len(in.Objects) == 0 {
		return DeleteObjectsOutput{}, errors.New("s3: no objects to delete")
	}
	if len(in.Objects) > 1000 {
		return DeleteObjectsOutput{}, errors.New("s3: cannot delete more than 1000 objects per request")
	}

	// 构造 XML 请求体
	payload := struct {
		XMLName xml.Name `xml:"Delete"`
		XMLNS   string   `xml:"xmlns,attr"`
		Quiet   bool     `xml:"Quiet"`
		Objects []struct {
			Key string `xml:"Key"`
		} `xml:"Object"`
	}{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Quiet: in.Quiet}
	for _, key := range in.Objects {
		payload.Objects = append(payload.Objects, struct {
			Key string `xml:"Key"`
		}{Key: key})
	}

	body, err := xml.Marshal(payload)
	if err != nil {
		return DeleteObjectsOutput{}, err
	}

	req, err := http.NewRequest(http.MethodPost, c.buildURL(in.Bucket, "")+"?delete", bytes.NewReader(body))
	if err != nil {
		return DeleteObjectsOutput{}, err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("x-amz-content-sha256", sha256Hex(body))
	// S3 对 DeleteObjects 强制要求 Content-MD5
	md5sum := md5.Sum(body)
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5sum[:]))

	respBody, _, err := c.do(req, http.StatusOK)
	if err != nil {
		return DeleteObjectsOutput{}, err
	}

	var result struct {
		Deleted []DeletedObject `xml:"Deleted"`
		Errors  []DeleteError   `xml:"Error"`
	}
	if err := xml.Unmarshal(respBody, &result); err != nil {
		return DeleteObjectsOutput{}, fmt.Errorf("s3: parse delete response: %w", err)
	}
	return DeleteObjectsOutput{Deleted: result.Deleted, Errors: result.Errors}, nil
}
