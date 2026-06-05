package docker

import (
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
)

// CopyLogs 将容器日志流写入 dst。
// 非 TTY 容器的日志是多路复用流（每帧带 8 字节头），需用 stdcopy 解复用；
// 传入同一个 writer 作为 stdout/stderr 两路目标，以保持原始时序。
// TTY 容器为裸流，直接拷贝即可。
func CopyLogs(dst io.Writer, src io.Reader, tty bool) error {
	if tty {
		_, err := io.Copy(dst, src)
		return err
	}
	_, err := stdcopy.StdCopy(dst, dst, src)
	return err
}
