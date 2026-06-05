package s3sdk

import (
	"encoding/xml"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
)

// ListInput 是 List / ListAll 的入参。
type ListInput struct {
	Bucket            string // bucket 名称
	Prefix            string // 可选：仅列举该前缀下的对象
	Delimiter         string // 可选：分组分隔符（通常为 "/"）
	MaxKeys           int64  // 可选：单页最大返回数（1-1000）
	ContinuationToken string // 可选：分页续传 token
	StartAfter        string // 可选：从该 key 之后开始列举
}

// Object 表示 bucket 中的一个对象。
type Object struct {
	Key          string
	Size         int64
	LastModified string
	ETag         string
	StorageClass string
}

// ListResponse 是单页 List 的返回。
type ListResponse struct {
	Name                  string
	IsTruncated           bool
	NextContinuationToken string
	Objects               []Object
	CommonPrefixes        []string
	KeyCount              int64
}

// List 调用 ListObjectsV2 列举单页对象。
func (c *S3) List(in ListInput) (ListResponse, error) {
	if in.MaxKeys < 0 || in.MaxKeys > 1000 {
		return ListResponse{}, fmt.Errorf("s3: MaxKeys must be between 0 and 1000, got %d", in.MaxKeys)
	}

	query := url.Values{"list-type": {"2"}}
	if in.Prefix != "" {
		query.Set("prefix", in.Prefix)
	}
	if in.Delimiter != "" {
		query.Set("delimiter", in.Delimiter)
	}
	if in.MaxKeys > 0 {
		query.Set("max-keys", strconv.FormatInt(in.MaxKeys, 10))
	}
	if in.ContinuationToken != "" {
		query.Set("continuation-token", in.ContinuationToken)
	}
	if in.StartAfter != "" {
		query.Set("start-after", in.StartAfter)
	}

	req, err := http.NewRequest(http.MethodGet, c.buildURL(in.Bucket, "")+"?"+query.Encode(), nil)
	if err != nil {
		return ListResponse{}, err
	}

	body, _, err := c.do(req, http.StatusOK)
	if err != nil {
		return ListResponse{}, err
	}

	var result struct {
		Name                  string   `xml:"Name"`
		IsTruncated           bool     `xml:"IsTruncated"`
		NextContinuationToken string   `xml:"NextContinuationToken"`
		KeyCount              int64    `xml:"KeyCount"`
		Contents              []Object `xml:"Contents"`
		CommonPrefixes        []struct {
			Prefix string `xml:"Prefix"`
		} `xml:"CommonPrefixes"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return ListResponse{}, fmt.Errorf("s3: parse list response: %w", err)
	}

	resp := ListResponse{
		Name:                  result.Name,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
		KeyCount:              result.KeyCount,
		Objects:               result.Contents,
	}
	for _, p := range result.CommonPrefixes {
		resp.CommonPrefixes = append(resp.CommonPrefixes, p.Prefix)
	}
	return resp, nil
}

// ListAll 返回一个自动翻页、产出 bucket 中全部对象的迭代器，
// 以及一个用于在迭代结束后获取错误的回调。
func (c *S3) ListAll(in ListInput) (iter.Seq[Object], func() error) {
	var iterErr error

	seq := func(yield func(Object) bool) {
		cur := in
		for {
			resp, err := c.List(cur)
			if err != nil {
				iterErr = err
				return
			}
			for _, obj := range resp.Objects {
				if !yield(obj) {
					return
				}
			}
			if !resp.IsTruncated || resp.NextContinuationToken == "" {
				return
			}
			cur.ContinuationToken = resp.NextContinuationToken
		}
	}

	return seq, func() error { return iterErr }
}
