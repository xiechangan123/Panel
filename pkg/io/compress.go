package io

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/pkg/shell"
)

type ArchiveFormat string

const (
	Zip      ArchiveFormat = "zip"
	SevenZip ArchiveFormat = "7z"
	Tar      ArchiveFormat = "tar"
	TGz      ArchiveFormat = "tgz"
	TBz2     ArchiveFormat = "tbz2"
	TXz      ArchiveFormat = "txz"
	TZst     ArchiveFormat = "tzst"
	Gz       ArchiveFormat = "gz"
	Bz2      ArchiveFormat = "bz2"
	Xz       ArchiveFormat = "xz"
	Zst      ArchiveFormat = "zst"
)

// Compress src 为空时压缩整个 dir
func Compress(ctx context.Context, dir string, src []string, dst string) error {
	cmd, err := CompressShell(dir, src, dst)
	if err != nil {
		return err
	}
	_, err = shell.Exec(ctx, cmd)
	return err
}

// UnCompress 解压到 dst 目录
func UnCompress(ctx context.Context, src, dst string) error {
	cmd, err := UnCompressShell(src, dst)
	if err != nil {
		return err
	}
	_, err = shell.Exec(ctx, cmd)
	return err
}

// CompressShell 生成压缩命令供后台任务执行
// 先删旧包，zip/7z 对已存在的目标是追加更新；tar/7z 退出码 1 只是警告（文件在打包时被改动、悬空链接），
// 容错要括在压缩命令自身上，否则前面 cd/rm 的失败也会被吞掉
func CompressShell(dir string, src []string, dst string) (string, error) {
	if !filepath.IsAbs(dir) || !filepath.IsAbs(dst) {
		return "", errors.New("dir and dst must be absolute path")
	}
	format, err := formatArchiveByPath(dst)
	if err != nil {
		return "", err
	}
	if len(src) == 0 {
		src = []string{"."}
	}
	target := shell.Quote(dst)
	sources := strings.Join(lo.Map(src, func(s string, _ int) string { return shell.Quote(s) }), " ")

	var cmd string
	switch format {
	case Zip:
		cmd = fmt.Sprintf("zip -qr %s -- %s", target, sources)
	case SevenZip:
		cmd = fmt.Sprintf("{ 7z a -y %s -- %s || [ $? -eq 1 ]; }", target, sources)
	case Tar, TGz, TBz2, TXz, TZst:
		cmd = fmt.Sprintf("{ tar -c %s -f %s -- %s || [ $? -eq 1 ]; }", tarFilter(format), target, sources)
	case Gz, Bz2, Xz, Zst:
		// 单文件压缩格式仅支持压缩单个文件
		if len(src) != 1 {
			return "", fmt.Errorf("%s format only supports compressing a single file", format)
		}
		cmd = fmt.Sprintf("%s -c -- %s > %s", compressor(format), sources, target)
	default:
		return "", errors.New("unsupported format")
	}

	return fmt.Sprintf("mkdir -p %s && cd %s && rm -f %s && %s", shell.Quote(filepath.Dir(dst)), shell.Quote(dir), target, cmd), nil
}

// UnCompressShell 生成解压命令供后台任务执行
func UnCompressShell(src, dst string) (string, error) {
	if !filepath.IsAbs(src) || !filepath.IsAbs(dst) {
		return "", errors.New("src and dst must be absolute path")
	}
	format, err := formatArchiveByPath(src)
	if err != nil {
		return "", err
	}
	source, target := shell.Quote(src), shell.Quote(dst)

	var cmd string
	switch format {
	case Zip, SevenZip:
		// 7-Zip 默认拒绝还原指向上级目录的符号链接（如 Laravel 的 public/storage），-snld 放行
		cmd = fmt.Sprintf("7z x -y -snld %s -o%s", source, target)
	case Tar, TGz, TBz2, TXz, TZst:
		cmd = fmt.Sprintf("tar -x %s -f %s -C %s", tarFilter(format), source, target)
	case Gz, Bz2, Xz, Zst:
		// 去掉压缩后缀落到目标目录
		name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		cmd = fmt.Sprintf("%s -dc -- %s > %s", compressor(format), source, shell.Quote(filepath.Join(dst, name)))
	default:
		return "", errors.New("unsupported format")
	}

	return fmt.Sprintf("mkdir -p %s && %s", target, cmd), nil
}

func tarFilter(format ArchiveFormat) string {
	switch format {
	case TGz:
		return "-z"
	case TBz2:
		return "-j"
	case TXz:
		return "-J"
	case TZst:
		return "--zstd"
	default:
		return ""
	}
}

func compressor(format ArchiveFormat) string {
	switch format {
	case Gz:
		return "gzip"
	case Bz2:
		return "bzip2"
	case Xz:
		return "xz"
	case Zst:
		return "zstd"
	default:
		return ""
	}
}

func formatArchiveByPath(path string) (ArchiveFormat, error) {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return TGz, nil
	case strings.HasSuffix(name, ".tar.bz2"), strings.HasSuffix(name, ".tbz2"):
		return TBz2, nil
	case strings.HasSuffix(name, ".tar.xz"), strings.HasSuffix(name, ".txz"):
		return TXz, nil
	case strings.HasSuffix(name, ".tar.zst"), strings.HasSuffix(name, ".tzst"):
		return TZst, nil
	}
	switch filepath.Ext(name) {
	case ".zip":
		return Zip, nil
	case ".7z":
		return SevenZip, nil
	case ".tar":
		return Tar, nil
	case ".gz":
		return Gz, nil
	case ".bz2":
		return Bz2, nil
	case ".xz":
		return Xz, nil
	case ".zst":
		return Zst, nil
	}

	return "", errors.New("unknown format")
}
