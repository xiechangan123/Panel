package s3sdk

import "testing"

func TestComputeBase(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"aws-vhost", Config{Region: "us-east-1", Bucket: "b"}, "https://b.s3.us-east-1.amazonaws.com"},
		{"aws-path", Config{Region: "us-east-1", Bucket: "b", PathStyle: true}, "https://s3.us-east-1.amazonaws.com/b"},
		{"endpoint-vhost", Config{Bucket: "b", Endpoint: "oss.example.com"}, "https://b.oss.example.com"},
		{"endpoint-path", Config{Bucket: "b", Endpoint: "oss.example.com", PathStyle: true}, "https://oss.example.com/b"},
		{"endpoint-scheme", Config{Bucket: "b", Endpoint: "http://oss.example.com", PathStyle: true}, "http://oss.example.com/b"},
		{"endpoint-trailing-slash", Config{Bucket: "b", Endpoint: "https://oss.example.com/", PathStyle: true}, "https://oss.example.com/b"},
	}
	for _, tc := range cases {
		if got := computeBase(tc.cfg); got != tc.want {
			t.Errorf("%s: computeBase() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
