package shell

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestQuoteRoundTrip(t *testing.T) {
	check.Equal(t, Quote("it's"), `'it'\''s'`)
	for _, s := range []string{"plain", "a b", "it's here", "$HOME `id` \"q\"", "-dash", "中文 文件"} {
		out, err := Exec(t.Context(), "printf %s "+Quote(s))
		must.NoError(t, err)
		check.Equal(t, out, s)
	}
}

func TestExecErrorCarriesStderr(t *testing.T) {
	out, err := Exec(t.Context(), "echo out; echo boom >&2; exit 3")
	check.Equal(t, out, "out")
	must.Error(t, err)
	check.True(t, strings.Contains(err.Error(), "boom"), check.Msgf("错误应带 stderr: %v", err))
	check.True(t, strings.Contains(err.Error(), "exit status 3"), check.Msgf("错误应带退出码: %v", err))
}

// 取消要连 bash 派生的孙子进程一起杀掉
func TestExecCancelKillsProcessGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()

	_, err := Exec(ctx, "sleep 30 & echo $! > "+Quote(pidFile)+"; wait")
	must.Error(t, err)

	raw, err := os.ReadFile(pidFile)
	must.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	must.NoError(t, err)
	// 被杀的孙子进程由 init 回收，给一点时间
	for range 20 {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("孙子进程 %d 仍然存活", pid)
}

// bash 退出后仍占着管道的后台进程只能等宽限期，到点后截断输出并报错
func TestExecWaitDelayTruncates(t *testing.T) {
	start := time.Now()
	out, err := Exec(t.Context(), "echo hi; sleep 6 &")
	check.Equal(t, out, "hi")
	check.True(t, errors.Is(err, exec.ErrWaitDelay), check.Msgf("应报宽限期超时: %v", err))
	check.True(t, time.Since(start) < 5*time.Second, check.Msgf("不应等到 sleep 结束"))
}

func TestExecfWithTimeout(t *testing.T) {
	_, err := ExecfWithTimeout(t.Context(), 200*time.Millisecond, "sleep 5")
	must.Error(t, err)
	check.True(t, strings.HasSuffix(err.Error(), "timeout"), check.Msgf("应报 timeout: %v", err))
}

func TestExecWithPipePropagatesFailure(t *testing.T) {
	r, err := ExecWithPipe(t.Context(), "echo partial; exit 2")
	must.NoError(t, err)
	var sb strings.Builder
	buf := make([]byte, 64)
	for {
		n, readErr := r.Read(buf)
		sb.Write(buf[:n])
		if readErr != nil {
			err = readErr
			break
		}
	}
	check.Equal(t, strings.TrimSpace(sb.String()), "partial")
	check.True(t, err != nil && strings.Contains(err.Error(), "exit status 2"), check.Msgf("读端应收到失败: %v", err))
}
