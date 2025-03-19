package data

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tnb-labs/panel/internal/app"
	"github.com/tnb-labs/panel/internal/biz"
	"github.com/tnb-labs/panel/pkg/shell"
)

type containerComposeRepo struct{}

func NewContainerComposeRepo() biz.ContainerComposeRepo {
	return &containerComposeRepo{}
}

// List 列出所有编排文件名
func (r *containerComposeRepo) List() ([]string, error) {
	dir := filepath.Join(app.Root, "server", "compose")
	var files []string
	err := filepath.Walk(dir, func(path string, d fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext == ".yml" || ext == ".yaml" {
			files = append(files, strings.TrimSuffix(filepath.Base(path), ext))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Get 获取编排文件内容
func (r *containerComposeRepo) Get(name string) (string, error) {
	dir := filepath.Join(app.Root, "server", "compose")
	path := filepath.Join(dir, name+".yml")
	content, err := os.ReadFile(path)
	return string(content), err
}

// Create 创建编排文件
func (r *containerComposeRepo) Create(name, compose string) error {
	dir := filepath.Join(app.Root, "server", "compose")
	path := filepath.Join(dir, name+".yml")
	return os.WriteFile(path, []byte(compose), 0644)
}

// Up 启动编排
func (r *containerComposeRepo) Up(name string, force bool) error {
	dir := filepath.Join(app.Root, "server", "compose")
	path := filepath.Join(dir, name+".yml")
	cmd := "docker compose -f %s up -d"
	if force {
		cmd += " --pull always" // 强制拉取镜像
	}
	_, err := shell.Execf(cmd, path)
	return err
}

// Down 停止编排
func (r *containerComposeRepo) Down(name string) error {
	dir := filepath.Join(app.Root, "server", "compose")
	path := filepath.Join(dir, name+".yml")
	_, err := shell.Execf("docker compose -f %s down", path)
	return err
}

// Remove 删除编排
func (r *containerComposeRepo) Remove(name string) error {
	dir := filepath.Join(app.Root, "server", "compose")
	path := filepath.Join(dir, name+".yml")
	return os.Remove(path)
}
