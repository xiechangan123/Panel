package s3sdk

import (
	"context"
	"encoding/xml"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"time"
)

// Object 是列举结果中的一个对象
type Object struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
}

// List 返回一个自动翻页、产出 prefix 下所有对象的迭代器
// delimiter 非空（通常为 "/"）时按目录层级分组
// 迭代中若出错，会以 (零值 Object, err) 产出一次随后结束
func (c *S3) List(prefix, delimiter string) iter.Seq2[Object, error] {
	return func(yield func(Object, error) bool) {
		token := ""
		for {
			page, err := c.listPage(prefix, delimiter, token)
			if err != nil {
				yield(Object{}, err)
				return
			}
			for _, obj := range page.objects {
				if !yield(obj, nil) {
					return
				}
			}
			if !page.truncated || page.nextToken == "" {
				return
			}
			token = page.nextToken
		}
	}
}

type listPage struct {
	objects   []Object
	truncated bool
	nextToken string
}

func (c *S3) listPage(prefix, delimiter, token string) (listPage, error) {
	query := url.Values{"list-type": {"2"}}
	if prefix != "" {
		query.Set("prefix", prefix)
	}
	if delimiter != "" {
		query.Set("delimiter", delimiter)
	}
	if token != "" {
		query.Set("continuation-token", token)
	}

	req, err := http.NewRequest(http.MethodGet, c.base+"?"+query.Encode(), nil)
	if err != nil {
		return listPage{}, err
	}

	body, _, err := c.do(context.Background(), req, http.StatusOK)
	if err != nil {
		return listPage{}, err
	}

	var result struct {
		IsTruncated           bool   `xml:"IsTruncated"`
		NextContinuationToken string `xml:"NextContinuationToken"`
		Contents              []struct {
			Key          string `xml:"Key"`
			Size         int64  `xml:"Size"`
			LastModified string `xml:"LastModified"`
			ETag         string `xml:"ETag"`
			StorageClass string `xml:"StorageClass"`
		} `xml:"Contents"`
	}
	if err := xml.Unmarshal(body, &result); err != nil {
		return listPage{}, fmt.Errorf("s3: parse list response: %w", err)
	}

	page := listPage{truncated: result.IsTruncated, nextToken: result.NextContinuationToken}
	for _, o := range result.Contents {
		t, _ := time.Parse(time.RFC3339, o.LastModified)
		page.objects = append(page.objects, Object{
			Key:          o.Key,
			Size:         o.Size,
			LastModified: t,
			ETag:         o.ETag,
			StorageClass: o.StorageClass,
		})
	}
	return page, nil
}
