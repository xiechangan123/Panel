package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"resty.dev/v3"
)

// DownloadFile 先落到同目录临时文件再原子替换，超时由 ctx 控制
func DownloadFile(ctx context.Context, url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*")
	if err != nil {
		return err
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()

	client := resty.New()
	defer func() { _ = client.Close() }()

	resp, err := client.R().
		SetContext(ctx).
		SetResponseSaveToFile(true).
		SetResponseSaveFileName(tmp.Name()).
		Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if resp.IsStatusFailure() {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	if err = os.Rename(tmp.Name(), dest); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
