//go:build linux

package scan

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

// TestSupportedAndAttach 验证检测程序可通过 verifier 且 TCX 可挂载
func TestSupportedAndAttach(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("需要 root")
	}
	if !Supported() {
		t.Skip("内核不支持 eBPF 扫描检测(TCX 需 6.6+)")
	}

	s, err := New([]string{"lo"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	t.Log("eBPF 扫描检测器: 程序加载 + TCX 挂载 + 端口同步 ✓")
}
