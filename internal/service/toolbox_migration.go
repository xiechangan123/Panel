package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/types"
)

// ToolboxMigrationService 面板迁移服务
type ToolboxMigrationService struct {
	t             *gotext.Locale
	conf          *config.Config
	log           *slog.Logger
	migrationRepo *biz.ToolboxMigrationUsecase
}

func NewToolboxMigrationService(
	migrationUsecase *biz.ToolboxMigrationUsecase,
	conf *config.Config,
	t *gotext.Locale,
	log *slog.Logger,
) (*ToolboxMigrationService, error) {
	return &ToolboxMigrationService{
		t: t, conf: conf, log: log, migrationRepo: migrationUsecase,
	}, nil
}

// PreCheck 连接来源面板并返回来源信息
func (s *ToolboxMigrationService) PreCheck(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ToolboxMigrationConnection](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	source, err := s.migrationRepo.Connect(r.Context(), req)
	if err != nil {
		Error(w, http.StatusBadGateway, "%v", err)
		return
	}

	Success(w, chix.M{"source": source})
}

// GetItems 获取可迁移资源列表
func (s *ToolboxMigrationService) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.migrationRepo.Items(r.Context())
	if err != nil {
		Error(w, http.StatusBadGateway, "%v", err)
		return
	}

	Success(w, chix.M{"items": items})
}

// Start 开始迁移
func (s *ToolboxMigrationService) Start(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ToolboxMigrationStart](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.migrationRepo.Start(req); err != nil {
		Error(w, http.StatusConflict, "%v", err)
		return
	}

	Success(w, nil)
}

// GetStatus 获取迁移状态与结果
func (s *ToolboxMigrationService) GetStatus(w http.ResponseWriter, r *http.Request) {
	step, results, logs, startedAt, endedAt := s.migrationRepo.Status(0)

	Success(w, chix.M{
		"step": step, "results": results, "logs": logs,
		"started_at": startedAt, "ended_at": endedAt,
	})
}

// Reset 重置迁移状态
func (s *ToolboxMigrationService) Reset(w http.ResponseWriter, r *http.Request) {
	if err := s.migrationRepo.Reset(); err != nil {
		Error(w, http.StatusConflict, "%v", err)
		return
	}

	Success(w, nil)
}

// DownloadLog 下载迁移日志
func (s *ToolboxMigrationService) DownloadLog(w http.ResponseWriter, r *http.Request) {
	logs := s.migrationRepo.Logs()
	if len(logs) == 0 {
		Error(w, http.StatusNotFound, s.t.Get("no migration logs available"))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=migration.log")
	_, _ = w.Write([]byte(strings.Join(logs, "\n")))
}

// Progress 通过 WebSocket 推送迁移进度
func (s *ToolboxMigrationService) Progress(w http.ResponseWriter, r *http.Request) {
	opts := &websocket.AcceptOptions{CompressionMode: websocket.CompressionContextTakeover}
	if s.conf.App.Debug {
		opts.InsecureSkipVerify = true
	}

	ws, err := websocket.Accept(w, r, opts)
	if err != nil {
		s.log.Warn("websocket upgrade error", slog.Any("err", err))
		return
	}
	defer func() { _ = ws.CloseNow() }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	sent := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			step, results, logs, startedAt, endedAt := s.migrationRepo.Status(sent)
			sent += len(logs)
			data, _ := json.Marshal(chix.M{
				"step": step, "results": results, "new_logs": logs,
				"started_at": startedAt, "ended_at": endedAt,
			})
			if err = ws.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
			if step == types.MigrationStepDone || step == types.MigrationStepIdle {
				_ = ws.Close(websocket.StatusNormalClosure, "")
				return
			}
		}
	}
}

// Exec SSE 实时执行命令，供来源面板推送迁移时调用
func (s *ToolboxMigrationService) Exec(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ToolboxMigrationExec](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	cmd := exec.CommandContext(r.Context(), "bash", "-c", req.Command)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err = cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	// 等待命令结束后关闭 pipe writer
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		_ = pw.Close()
	}()

	// 命令报错时单行输出可能很长（如数据库导入错误附带语句内容），加大行缓冲
	scanner := bufio.NewScanner(pr)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}

	if waitErr := <-waitCh; waitErr != nil {
		_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", waitErr.Error())
	} else {
		_, _ = fmt.Fprintf(w, "event: done\ndata: ok\n\n")
	}
	flusher.Flush()
}
