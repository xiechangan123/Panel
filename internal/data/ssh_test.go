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

	// 目录递归传输:重建目录树、内容一致、进度按全树累计
	tree := filepath.Join(dir, "tree")
	if err = os.MkdirAll(filepath.Join(tree, "a/b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(tree, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(tree, "a/b/deep.txt"), []byte("deep"), 0600); err != nil {
		t.Fatal(err)
	}
	treeDst := filepath.Join(dir, "tree-copy")
	var treeTransferred, treeTotal int64
	if err = repo.TransferFile(context.Background(), 0, tree, 0, treeDst, func(transferred, total int64) {
		treeTransferred, treeTotal = transferred, total
	}); err != nil {
		t.Fatalf("transfer directory: %v", err)
	}
	if treeTransferred != 8 || treeTotal != 8 {
		t.Fatalf("directory progress mismatch: transferred=%d total=%d", treeTransferred, treeTotal)
	}
	for name, want := range map[string]string{"root.txt": "root", "a/b/deep.txt": "deep"} {
		got, err := os.ReadFile(filepath.Join(treeDst, name))
		if err != nil || string(got) != want {
			t.Fatalf("directory content mismatch for %s: %v", name, err)
		}
	}
	if stat, _ := os.Stat(filepath.Join(treeDst, "a/b/deep.txt")); stat.Mode().Perm() != 0600 {
		t.Fatalf("directory file mode not preserved: %v", stat.Mode())
	}

	// 取消传输
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = repo.TransferFile(ctx, 0, src, 0, filepath.Join(dir, "cancelled.bin"), func(int64, int64) {}); err == nil {
		t.Fatal("expected error for cancelled transfer")
	}
}
