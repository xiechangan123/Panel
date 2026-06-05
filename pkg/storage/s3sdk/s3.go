package s3sdk

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

// AMZMetaPrefix 是对象自定义元数据 header 的前缀。
const AMZMetaPrefix = "x-amz-meta-"

// S3 封装 S3 访问凭证与端点配置。
type S3 struct {
	AccessKey string
	SecretKey string
	Region    string
	Client    *http.Client // 为空时使用 http.DefaultClient

	Endpoint  string // 自定义端点；为空时使用 AWS 默认端点
	PathStyle bool   // true 为 path-style，false 为 virtual-hosted style
}

// New 返回一个使用 path-style 寻址的 S3 客户端。
func New(region, accessKey, secretKey string) *S3 {
	return &S3{
		Region:    region,
		AccessKey: accessKey,
		SecretKey: secretKey,
		PathStyle: true,
	}
}

// SetEndpoint 设置兼容 S3 API 的自定义端点；未带协议时默认 HTTPS。
func (c *S3) SetEndpoint(endpoint string) *S3 {
	if endpoint != "" {
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "https://" + endpoint
		}
		c.Endpoint = strings.TrimRight(endpoint, "/")
	}
	return c
}

// SetVirtualHostedStyle 启用 virtual-hosted 风格寻址
// （形如 https://bucket.s3.region.amazonaws.com/key）。
// 此时 bucket 名应包含在 Endpoint 中，调用各方法时 Bucket 参数留空。
func (c *S3) SetVirtualHostedStyle() *S3 {
	c.PathStyle = false
	return c
}

func (c *S3) httpClient() *http.Client {
	if c.Client == nil {
		return http.DefaultClient
	}
	return c.Client
}

// buildURL 根据端点配置为 bucket/key 构造请求 URL，key 为空时只到 bucket 层级。
func (c *S3) buildURL(bucket, key string) string {
	path := bucket
	if key != "" {
		if path != "" {
			path += "/" + key
		} else {
			path = key
		}
	}
	path = encodePath(path)

	if c.Endpoint != "" {
		if path == "" {
			return c.Endpoint
		}
		return c.Endpoint + "/" + path
	}
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s", c.Region, path)
}

// do 对请求签名并执行，读取完整响应体；当状态码不等于 wantStatus 时返回 *apiError。
// 返回的 *http.Response 仅用于读取响应头，其 Body 已被读尽并关闭。
func (c *S3) do(req *http.Request, wantStatus int) ([]byte, *http.Response, error) {
	c.signRequest(req)

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res, err
	}
	if res.StatusCode != wantStatus {
		return body, res, &apiError{status: res.StatusCode, statusText: res.Status, body: body}
	}
	return body, res, nil
}

// apiError 表示一个非预期状态码的 S3 响应。
type apiError struct {
	status     int
	statusText string
	body       []byte
}

func (e *apiError) Error() string {
	// 优先解析 S3 的 XML 错误体以给出更友好的信息
	var x struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	if xml.Unmarshal(e.body, &x) == nil && x.Code != "" {
		return fmt.Sprintf("s3: %s: %s", x.Code, x.Message)
	}
	if trimmed := bytes.TrimSpace(e.body); len(trimmed) > 0 {
		return fmt.Sprintf("s3: unexpected status %s: %s", e.statusText, trimmed)
	}
	return fmt.Sprintf("s3: unexpected status %s", e.statusText)
}

// 仅由未保留字符组成的对象名无需编码。
var reservedObjectNames = regexp.MustCompile(`^[a-zA-Z0-9\-_.~/]+$`)

// encodePath 将路径按 RFC3986 对每个 UTF-8 字符做百分号编码，但保留 / 等未保留字符。
// 标准库 url 包无法正确处理路径中的非 ASCII 字符，故自行实现。
// 改编自 minio-go 的 s3utils.EncodePath。
func encodePath(pathName string) string {
	if reservedObjectNames.MatchString(pathName) {
		return pathName
	}
	var b strings.Builder
	for _, r := range pathName {
		switch {
		case 'A' <= r && r <= 'Z', 'a' <= r && r <= 'z', '0' <= r && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.', r == '~', r == '/':
			b.WriteRune(r)
		default:
			n := utf8.RuneLen(r)
			if n < 0 {
				return pathName // 无法编码则原样返回
			}
			buf := make([]byte, n)
			utf8.EncodeRune(buf, r)
			for _, c := range buf {
				b.WriteString("%" + strings.ToUpper(hex.EncodeToString([]byte{c})))
			}
		}
	}
	return b.String()
}
