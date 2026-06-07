package s3sdk

import (
	"net/http"
	"testing"
	"time"
)

// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
func TestSignRequestAWSExample(t *testing.T) {
	c := &S3{
		region:    "us-east-1",
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	req, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-9")
	req.Header.Set("x-amz-content-sha256", emptyPayloadSHA256)

	c.signRequestAt(req, time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC))

	const want = "AWS4-HMAC-SHA256 " +
		"Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, " +
		"SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, " +
		"Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got := req.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization 不匹配 AWS 官方向量:\n got: %s\nwant: %s", got, want)
	}
}

func TestURIEncode(t *testing.T) {
	cases := []struct {
		in    string
		slash bool
		want  string
	}{
		{"abc", true, "abc"},
		{"a b", true, "a%20b"},   // 空格须编码为 %20 而非 +
		{"~", true, "~"},         // 波浪号属未保留字符，不编码
		{"a+b", true, "a%2Bb"},   // 加号须编码
		{"a/b", false, "a/b"},    // 路径模式保留斜杠
		{"a/b", true, "a%2Fb"},   // 查询模式编码斜杠
		{"中", true, "%E4%B8%AD"}, // UTF-8 逐字节编码
	}
	for _, tc := range cases {
		if got := uriEncode(tc.in, tc.slash); got != tc.want {
			t.Errorf("uriEncode(%q, %v) = %q, want %q", tc.in, tc.slash, got, tc.want)
		}
	}
}
