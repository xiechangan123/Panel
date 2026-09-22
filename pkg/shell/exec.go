package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// waitDelay 组 kill 之后等待 stdout 管道关闭的宽限期
// 用 setsid 逃出进程组的孙子进程杀不掉，它占着管道写端会让 Wait 永不返回，超过宽限期就截断输出并报错
const waitDelay = 3 * time.Second

// Command 构造受 ctx 控制的命令
// 子进程还会 fork（bash 遇到管道和 &&、tar 起 gzip、用户脚本起守护进程），CommandContext 默认只 Kill 它本身，
// 孙子进程不但活着还会拖住 Wait，因此统一放进独立进程组整组杀死
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	applyEnv(cmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay

	return cmd
}

func applyEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "LC_ALL=C")
}

func newCmd(ctx context.Context, shell string) *exec.Cmd {
	return Command(ctx, "bash", "-c", shell)
}

// buildShell 格式化命令，参数由调用方用 Quote 处理
func buildShell(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// runBuffered 执行命令并返回 stdout，失败时附带 stderr
func runBuffered(cmd *exec.Cmd, shell string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(stdout.String()), fmt.Errorf("run %s failed, err: %w, stderr: %s", shell, err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Exec 执行已拼好的 shell 命令
func Exec(ctx context.Context, shell string) (string, error) {
	return runBuffered(newCmd(ctx, shell), shell)
}

// Execf 按格式串拼出 shell 命令后执行
func Execf(ctx context.Context, format string, args ...any) (string, error) {
	return Exec(ctx, buildShell(format, args...))
}

// ExecfWithEnv 环境变量仅注入子进程
func ExecfWithEnv(ctx context.Context, env []string, format string, args ...any) (string, error) {
	shell := buildShell(format, args...)
	cmd := newCmd(ctx, shell)
	cmd.Env = append(cmd.Env, env...)

	return runBuffered(cmd, shell)
}

// ExecfWithDir 在指定目录下执行
func ExecfWithDir(ctx context.Context, dir, format string, args ...any) (string, error) {
	shell := buildShell(format, args...)
	cmd := newCmd(ctx, shell)
	cmd.Dir = dir

	return runBuffered(cmd, shell)
}

// ExecfWithTimeout ctx 取消或超时到期均终止进程
func ExecfWithTimeout(ctx context.Context, timeout time.Duration, format string, args ...any) (string, error) {
	shell := buildShell(format, args...)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := runBuffered(newCmd(timeoutCtx, shell), shell)
	// 只有本函数的 timeout 到期才算超时，父 ctx 自带的 deadline 要如实上报
	if err != nil && ctx.Err() == nil && errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("run %s failed, err: timeout", shell)
	}

	return out, err
}

// ExecfAsync 异步执行，命令要活过调用方
// 入口断开取消链，否则 sleep 1 && systemctl restart acepanel 这类自杀式操作会在 HTTP 响应写完的瞬间被杀且无人察觉
func ExecfAsync(ctx context.Context, format string, args ...any) error {
	shell := buildShell(format, args...)
	cmd := Command(context.WithoutCancel(ctx), "bash", "-c", shell)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Warn("async command failed", slog.String("cmd", shell), slog.Any("err", err))
		}
	}()

	return nil
}

// ExecWithOutput 执行已拼好的 shell 命令并输出到终端
func ExecWithOutput(ctx context.Context, shell string) error {
	cmd := newCmd(ctx, shell)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ExecWithPipe 执行已拼好的 shell 命令，stdout 与 stderr 合并成流返回，命令失败时读端会收到错误
func ExecWithPipe(ctx context.Context, shell string) (io.ReadCloser, error) {
	cmd := newCmd(ctx, shell)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		_, _ = io.Copy(pw, stdout)
		_ = pw.CloseWithError(cmd.Wait())
	}()

	return pr, nil
}

// ExecWithLog 输出覆盖写入日志文件
func ExecWithLog(ctx context.Context, shell string, logFile string) error {
	return execWithLog(ctx, shell, logFile, os.O_TRUNC)
}

// ExecWithLogAppend 输出追加到日志文件
func ExecWithLogAppend(ctx context.Context, shell string, logFile string) error {
	return execWithLog(ctx, shell, logFile, os.O_APPEND)
}

func execWithLog(ctx context.Context, shell string, logFile string, flag int) error {
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|flag, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	cmd := newCmd(ctx, shell)
	cmd.Stdout = f
	cmd.Stderr = f

	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("run shell failed: %w", err)
	}

	return nil
}
