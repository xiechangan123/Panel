package data

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

type migrationArchiveRepo struct{}

func NewMigrationArchiveRepo() (biz.MigrationArchiveRepo, error) {
	return &migrationArchiveRepo{}, nil
}

// TempDir 在面板目录下创建临时目录，避免大文件撑爆 /tmp（常为内存盘）
func (r *migrationArchiveRepo) TempDir() (string, error) {
	base := filepath.Join(app.Root, "tmp")
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "migration-*")
}

// Extract 解包归档，返回真实内容根目录（归档常带一层同名目录）
func (r *migrationArchiveRepo) Extract(ctx context.Context, archive, target string) (string, error) {
	if err := os.MkdirAll(target, 0755); err != nil {
		return "", err
	}
	if output, err := exec.CommandContext(ctx, "tar", "-xf", archive, "-C", target).CombinedOutput(); err != nil {
		return "", fmt.Errorf("extract archive: %s", strings.TrimSpace(string(output)))
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return target, nil
	}
	return filepath.Join(target, entries[0].Name()), nil
}

// Compress 打包目录为 tar.gz
func (r *migrationArchiveRepo) Compress(ctx context.Context, source, archive string) error {
	if err := os.MkdirAll(filepath.Dir(archive), 0755); err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, "tar", "-czf", archive, "-C", source, ".").CombinedOutput()
	if err != nil {
		return fmt.Errorf("compress directory: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// CopyTree 复制目录内容，保留权限与属主
func (r *migrationArchiveRepo) CopyTree(ctx context.Context, source, target string) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	// 必须以 /. 结尾才是复制目录内容，filepath.Join 会把尾部的 . 清理掉，导致源目录被整个拷进目标
	output, err := exec.CommandContext(ctx, "cp", "-a", source+"/.", target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy directory: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// IsEmpty 判断目录为空或不存在
func (r *migrationArchiveRepo) IsEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	return err != nil || len(entries) == 0
}
