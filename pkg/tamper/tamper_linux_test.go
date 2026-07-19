//go:build linux

package tamper

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/acepanel/panel/v3/pkg/chattr"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// tryWrite 尝试以写方式打开文件,返回是否被拒绝
func tryWrite(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return true // 被拒
	}
	_ = f.Close()
	return false
}

// chattrSupported 检测当前文件系统是否支持 immutable 属性
func chattrSupported(dir string) bool {
	p := filepath.Join(dir, ".chattr_probe")
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		return false
	}
	defer os.Remove(p)
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	if err := chattr.SetAttr(f, chattr.FS_IMMUTABLE_FL); err != nil {
		return false
	}
	_ = chattr.UnsetAttr(f, chattr.FS_IMMUTABLE_FL)
	return true
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestChattrMode(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("需要 root")
	}
	dir := t.TempDir()
	if !chattrSupported(dir) {
		t.Skip("文件系统不支持 immutable 属性")
	}

	protected := filepath.Join(dir, "index.html")
	free := filepath.Join(dir, "cache.txt")
	writeFile(t, protected, "original")
	writeFile(t, free, "cache")

	m, err := NewManager(Config{
		Mode:  ModeChattr,
		Rules: []Rule{{Name: "test", Paths: []string{dir}, Exts: []string{"html"}}},
	}, newLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}

	if !tryWrite(protected) {
		m.Stop()
		t.Fatal("受保护文件应写入被拒")
	}
	if tryWrite(free) {
		m.Stop()
		t.Fatal("非保护文件应可写")
	}

	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	if tryWrite(protected) {
		t.Fatal("停止后受保护文件应恢复可写")
	}
	t.Log("chattr 模式: 保护生效、非保护放行、停止后恢复 ✓")
}

func TestEBPFMode(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("需要 root")
	}
	st := DetectEBPF()
	if !st.Available {
		t.Skipf("eBPF 不可用: %s (lsm=%s)", st.Reason, st.ActiveLSM)
	}
	t.Logf("内核 %s, LSM=%s", st.KernelVersion, st.ActiveLSM)

	dir := t.TempDir()
	protected := filepath.Join(dir, "index.php")
	free := filepath.Join(dir, "upload.txt")
	victim := filepath.Join(dir, "del.php")
	writeFile(t, protected, "<?php original")
	writeFile(t, free, "upload")
	writeFile(t, victim, "<?php delete-me")

	m, err := NewManager(Config{
		Mode:  ModeEBPF,
		Rules: []Rule{{Name: "test", Paths: []string{dir}, Exts: []string{"php"}}},
	}, newLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	// 写拦截
	if !tryWrite(protected) {
		t.Fatal("受保护 .php 应写入被拒")
	}
	// 非保护放行
	if tryWrite(free) {
		t.Fatal("非保护 .txt 应可写")
	}
	// 删除拦截
	if err := os.Remove(victim); err == nil {
		t.Fatal("受保护 .php 应删除被拒")
	}
	// 重命名拦截
	if err := os.Rename(protected, filepath.Join(dir, "renamed.php")); err == nil {
		t.Fatal("受保护 .php 应重命名被拒")
	}
	// 改名覆盖拦截(rename 到受保护目标,经 new_dentry 检查)
	evil := filepath.Join(dir, "evil.txt")
	writeFile(t, evil, "evil")
	if err := os.Rename(evil, victim); err == nil {
		t.Fatal("改名覆盖受保护 .php 应被拒")
	}
	// 截断拦截(truncate 不经写打开,经 inode_setattr)
	if err := os.Truncate(protected, 0); err == nil {
		t.Fatal("受保护 .php 应截断被拒")
	}
	// 属性修改拦截
	if err := os.Chmod(protected, 0777); err == nil {
		t.Fatal("受保护 .php 应 chmod 被拒")
	}

	// 事件回填路径
	select {
	case ev := <-m.Events():
		if ev.Path == "" {
			t.Error("事件应回填路径")
		}
		t.Logf("拦截事件: op=%s path=%s comm=%s pid=%d", ev.OpStr, ev.Path, ev.Comm, ev.PID)
	case <-time.After(2 * time.Second):
		t.Error("应收到拦截事件")
	}

	// 解锁后可写,恢复后再拒
	m.Unlock([]string{protected})
	if tryWrite(protected) {
		t.Error("解锁后应可写")
	}
	m.Relock([]string{protected})
	if !tryWrite(protected) {
		t.Error("恢复后应再次被拒")
	}

	t.Log("eBPF 模式: 写/删/改名/覆盖/截断/属性拦截、事件回填、解锁恢复 ✓")
}

func TestScanExcludeAndExt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.php"), "1")
	writeFile(t, filepath.Join(dir, "b.txt"), "2")
	_ = os.MkdirAll(filepath.Join(dir, "cache"), 0755)
	writeFile(t, filepath.Join(dir, "cache", "c.php"), "3")

	m := &Manager{cfg: Config{
		Rules: []Rule{{Paths: []string{dir}, Exts: []string{"php"}, Excludes: []string{"cache"}}},
	}}
	entries := m.scan()

	var got []string
	for _, e := range entries {
		got = append(got, filepath.Base(e.path))
	}
	if len(got) != 1 || got[0] != "a.php" {
		t.Fatalf("期望仅 a.php,实得 %v", got)
	}
	// inode 应有效
	var st syscall.Stat_t
	_ = syscall.Lstat(filepath.Join(dir, "a.php"), &st)
	if entries[0].inode != st.Ino {
		t.Fatalf("inode 不匹配")
	}
	t.Log("扫描: 后缀过滤 + 排除目录 + inode 采集 ✓")
}
