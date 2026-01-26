package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/crypto/ssh"
)

type MessageResize struct {
	Resize  bool `json:"resize"`
	Columns int  `json:"columns"`
	Rows    int  `json:"rows"`
}

type MessagePing struct {
	Ping bool `json:"ping"`
}

type Turn struct {
	ctx     context.Context
	stdin   io.WriteCloser
	session *ssh.Session
	ws      *websocket.Conn
}

func NewTurn(ctx context.Context, ws *websocket.Conn, client *ssh.Client) (*Turn, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	stdin, err := sess.StdinPipe()
	if err != nil {
		return nil, err
	}

	turn := &Turn{ctx: ctx, stdin: stdin, session: sess, ws: ws}
	sess.Stdout = turn
	sess.Stderr = turn

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err = sess.RequestPty("xterm", 150, 80, modes); err != nil {
		return nil, err
	}
	if err = sess.Shell(); err != nil {
		return nil, err
	}

	return turn, nil
}

func (t *Turn) Write(p []byte) (n int, err error) {
	if err = t.ws.Write(t.ctx, websocket.MessageText, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *Turn) Close() {
	_ = t.stdin.Close()
	_ = t.session.Signal(ssh.SIGTERM)
	// 等待 10 秒
	time.Sleep(10 * time.Second)
	_ = t.session.Signal(ssh.SIGKILL)
	_ = t.session.Close()
}

func (t *Turn) Handle(ctx context.Context) error {
	var resize MessageResize
	var ping MessagePing

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, data, err := t.ws.Read(ctx)
			if err != nil {
				// 通常是客户端关闭连接
				return fmt.Errorf("reading ws message err: %v", err)
			}

			// 判断是否是 ping 消息
			if err = json.Unmarshal(data, &ping); err == nil && ping.Ping {
				_ = t.ws.Write(ctx, websocket.MessageText, []byte(`{"pong":true}`))
				continue
			}

			// 判断是否是 resize 消息
			if err = json.Unmarshal(data, &resize); err == nil {
				if resize.Resize && resize.Columns > 0 && resize.Rows > 0 {
					if err = t.session.WindowChange(resize.Rows, resize.Columns); err != nil {
						return fmt.Errorf("change window size err: %v", err)
					}
				}
				continue
			}

			if _, err = t.stdin.Write(data); err != nil {
				return fmt.Errorf("writing ws message to stdin err: %v", err)
			}
		}
	}
}

func (t *Turn) Wait() {
	_ = t.session.Wait()
}
