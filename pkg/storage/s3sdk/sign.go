package s3sdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	algorithm       = "AWS4-HMAC-SHA256"
	serviceName     = "s3"
	amzDateFormat   = "20060102T150405Z"
	shortDateFormat = "20060102"
	// emptyPayloadSHA256 是空请求体的 SHA-256，SigV4 在无 body 时要求使用该值。
	emptyPayloadSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// signRequest 按 AWS Signature V4 为请求添加鉴权头。
// 约定：调用方若有请求体，须先设置 x-amz-content-sha256 头为其 SHA-256；
// 未设置时按空 body 处理。
func (c *S3) signRequest(req *http.Request) {
	c.signRequestAt(req, time.Now().UTC())
}

// signRequestAt 是 signRequest 的可测试内核，使用给定时间签名。
func (c *S3) signRequestAt(req *http.Request, now time.Time) {
	amzDate := now.Format(amzDateFormat)

	req.Header.Set("Host", req.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	if req.Header.Get("x-amz-content-sha256") == "" {
		req.Header.Set("x-amz-content-sha256", emptyPayloadSHA256)
	}

	scope := now.Format(shortDateFormat) + "/" + c.Region + "/" + serviceName + "/aws4_request"
	signed := signedHeaders(req)

	// 规范请求（Canonical Request）
	canonical := strings.Join([]string{
		req.Method,
		canonicalURI(req),
		canonicalQuery(req),
		canonicalHeaders(req), // 每行以 \n 结尾
		signed,
		req.Header.Get("x-amz-content-sha256"),
	}, "\n")

	// 待签字符串（String to Sign）
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		scope,
		sha256Hex([]byte(canonical)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(c.signingKey(now), []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, c.AccessKey, scope, signed, signature,
	))
}

// signingKey 派生 SigV4 签名密钥：key = HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), service), "aws4_request")。
func (c *S3) signingKey(t time.Time) []byte {
	k := hmacSHA256([]byte("AWS4"+c.SecretKey), []byte(t.Format(shortDateFormat)))
	k = hmacSHA256(k, []byte(c.Region))
	k = hmacSHA256(k, []byte(serviceName))
	return hmacSHA256(k, []byte("aws4_request"))
}

// canonicalURI 返回去除查询串后、经清理的资源路径（已是 URL 编码形式）。
func canonicalURI(req *http.Request) string {
	p := req.URL.RequestURI()
	if req.URL.RawQuery != "" {
		p = p[:len(p)-len(req.URL.RawQuery)-1]
	}
	trailing := strings.HasSuffix(p, "/")
	p = path.Clean(p) // 必须用 path.Clean 而非 filepath.Clean，后者在 Windows 上会用反斜杠
	if p != "/" && trailing {
		p += "/"
	}
	return p
}

// canonicalQuery 返回按键、值排序并编码的查询串。
func canonicalQuery(req *http.Request) string {
	query := req.URL.Query()
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		values := query[k]
		sort.Strings(values)
		for _, v := range values {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}

// canonicalHeaders 返回按名（小写）排序的规范头，每行形如 "name:value\n"。
func canonicalHeaders(req *http.Request) string {
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		values := req.Header.Values(http.CanonicalHeaderKey(k))
		sort.Strings(values)
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(strings.Join(values, ","))
		b.WriteByte('\n')
	}
	return b.String()
}

// signedHeaders 返回参与签名的头名列表，按小写排序、以分号连接。
func signedHeaders(req *http.Request) string {
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
