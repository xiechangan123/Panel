package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"
)

// waitDelay 组 kill 之后等待 stdout 管道关闭的宽限期
// 用 setsid 逃出进程组的孙子进程杀不掉，它占着管道写端会让 Wait 永不返回，超过宽限期就截断输出并报错
const waitDelay = 3 * time.Second

func ApplyEnv(cmd *exec.Cmd, env ...string) {
	cmd.Env = append(os.Environ(), append([]string{"LC_ALL=C"}, env...)...)
}

// newCmd 构造受 ctx 控制的 bash 命令
// bash 遇到管道和 && 会 fork，CommandContext 默认只 Kill bash 本身，孙子进程不但活着还会拖住 Wait，
// 因此统一放进独立进程组整组杀死
func newCmd(ctx context.Context, shell string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "bash", "-c", shell)
	ApplyEnv(cmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay

	return cmd
}

// buildShell 校验参数并格式化命令
func buildShell(shell string, args []any) (string, error) {
	if !preCheckArg(args) {
		return "", errors.New("command contains illegal characters")
	}
	if len(args) > 0 {
		shell = fmt.Sprintf(shell, args...)
	}

	return shell, nil
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

// Execf 安全执行 shell 命令
func Execf(ctx context.Context, shell string, args ...any) (string, error) {
	shell, err := buildShell(shell, args)
	if err != nil {
		return "", err
	}

	return runBuffered(newCmd(ctx, shell), shell)
}

// ExecfWithEnv 安全执行 shell 命令，环境变量仅注入子进程
func ExecfWithEnv(ctx context.Context, env []string, shell string, args ...any) (string, error) {
	shell, err := buildShell(shell, args)
	if err != nil {
		return "", err
	}

	cmd := newCmd(ctx, shell)
	ApplyEnv(cmd, env...)

	return runBuffered(cmd, shell)
}

// ExecfWithDir 在指定目录下执行 shell 命令
func ExecfWithDir(ctx context.Context, dir, shell string, args ...any) (string, error) {
	shell, err := buildShell(shell, args)
	if err != nil {
		return "", err
	}

	cmd := newCmd(ctx, shell)
	cmd.Dir = dir

	return runBuffered(cmd, shell)
}

// ExecfWithTimeout 执行 shell 命令并设置超时时间，ctx 取消或超时到期均终止进程
func ExecfWithTimeout(ctx context.Context, timeout time.Duration, shell string, args ...any) (string, error) {
	shell, err := buildShell(shell, args)
	if err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := runBuffered(newCmd(timeoutCtx, shell), shell)
	// 只有本函数的 timeout 到期才算超时，父 ctx 自带的 deadline 要如实上报
	if err != nil && ctx.Err() == nil && errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("run %s failed, err: timeout", shell)
	}

	return out, err
}

// ExecfAsync 异步执行 shell 命令
// 异步即命令要活过调用方，入口断开取消链，否则 sleep 1 && systemctl restart acepanel 这类自杀式操作
// 会在 HTTP 响应写完的瞬间被杀且无人察觉
func ExecfAsync(ctx context.Context, shell string, args ...any) error {
	shell, err := buildShell(shell, args)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(context.WithoutCancel(ctx), "bash", "-c", shell)
	ApplyEnv(cmd)
	// 独立进程组，面板自身被信号终止时不牵连它
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err = cmd.Start(); err != nil {
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Println(fmt.Errorf("run %s failed, err: %s", shell, strings.TrimSpace(err.Error())))
		}
	}()

	return nil
}

// ExecfWithOutput 执行 shell 命令并输出到终端
func ExecfWithOutput(ctx context.Context, shell string, args ...any) error {
	shell, err := buildShell(shell, args)
	if err != nil {
		return err
	}

	cmd := newCmd(ctx, shell)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ExecfWithPipe 执行 shell 命令并返回管道
func ExecfWithPipe(ctx context.Context, shell string, args ...any) (io.ReadCloser, error) {
	shell, err := buildShell(shell, args)
	if err != nil {
		return nil, err
	}

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
		_ = cmd.Wait()
		_ = pw.Close()
	}()

	return pr, nil
}

// ExecWithLog 执行 shell 命令并将输出覆盖写入指定的日志文件
func ExecWithLog(ctx context.Context, shell string, logFile string) error {
	return execWithLog(ctx, shell, logFile, os.O_TRUNC)
}

// ExecWithLogAppend 执行 shell 命令并将输出追加到指定的日志文件
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

func preCheckArg(args []any) bool {
	illegals := []any{`&`, `|`, `;`, `$`, `'`, `"`, "`", `(`, `)`, "\n", "\r", `>`, `<`}
	for arg := range slices.Values(args) {
		if slices.Contains(illegals, arg) {
			return false
		}
	}

	return true
}
