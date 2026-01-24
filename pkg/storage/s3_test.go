package storage

import "testing"

func TestS3(t *testing.T) {
	s3, err := NewS3(S3Config{
		Region: "us-west-1",
	})
}
