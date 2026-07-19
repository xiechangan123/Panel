package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/leonelquinteros/gotext"
)

func TestSSHLocalFileOps(t *testing.T) {
	repo := &sshRepo{t: gotext.NewLocale("", "en"), conns: make(map[uint]*sftpConn)}
	dir := t.TempDir()

	// 准备源文件
	data := make([]byte, 2<<20)
	_, _ = rand.Read(data)
	src := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(src, data, 0640); err != nil {
		t.Fatal(err)
	}

	// 列目录
	files, err := repo.ListFiles(0, dir)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0].Name != "src.bin" || files[0].Size != int64(len(data)) {
		t.Fatalf("unexpected files: %+v", files[0])
	}

	// 创建目录
	sub := filepath.Join(dir, "sub/deep")
	if err = repo.Mkdir(0, sub); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// 本机到本机传输
	dst := filepath.Join(sub, "dst.bin")
	var calls int
	var lastTransferred, lastTotal int64
	err = repo.TransferFile(context.Background(), 0, src, 0, dst, func(transferred, total int64) {
		calls++
		lastTransferred, lastTotal = transferred, total
	})
	if err != nil {
		t.Fatalf("TransferFile: %v", err)
	}
	if calls == 0 || lastTransferred != int64(len(data)) || lastTotal != int64(len(data)) {
		t.Fatalf("progress not reported: calls=%d transferred=%d total=%d", calls, lastTransferred, lastTotal)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("content mismatch")
	}
	if stat, _ := os.Stat(dst); stat.Mode().Perm() != 0640 {
		t.Fatalf("mode not preserved: %v", stat.Mode())
	}

	// 目录拒绝传输
	if err = repo.TransferFile(context.Background(), 0, sub, 0, filepath.Join(dir, "x"), func(int64, int64) {}); err == nil {
		t.Fatal("expected error for directory transfer")
	}

	// 取消传输
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = repo.TransferFile(ctx, 0, src, 0, filepath.Join(dir, "cancelled.bin"), func(int64, int64) {}); err == nil {
		t.Fatal("expected error for cancelled transfer")
	}
}
