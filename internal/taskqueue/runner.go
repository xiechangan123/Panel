package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/shell"
)

type Runner struct {
	db     *gorm.DB
	log    *slog.Logger
	notify chan struct{}

	mu            sync.Mutex
	currentID     uint               // 当前运行的任务 ID
	currentCancel context.CancelFunc // 取消当前任务
}

// NewRunner 创建任务运行器
func NewRunner(db *gorm.DB, log *slog.Logger) *Runner {
	return &Runner{
		db:     db,
		log:    log,
		notify: make(chan struct{}, 1),
	}
}

// Notify 非阻塞通知运行器有新任务，供 Push 调用
func (r *Runner) Notify() {
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

// Cancel 取消正在运行的任务，返回是否命中
func (r *Runner) Cancel(id uint) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.currentID != id || r.currentCancel == nil {
		return false
	}
	r.currentCancel()
	return true
}

// Run 启动运行器
func (r *Runner) Run(ctx context.Context) {
	go func() {
		r.clearZombie()

		// 启动时先尝试处理积压的 waiting 任务
		r.Notify()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-r.notify:
				r.drain(ctx)
			case <-ticker.C:
				r.drain(ctx)
			}
		}
	}()
}

// drain 持续处理 waiting 任务直到队列为空或 ctx 取消
func (r *Runner) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if !r.processNext(ctx) {
			return
		}
	}
}

// clearZombie 启动时将残留的 running 任务标记为 failed
func (r *Runner) clearZombie() {
	if err := r.db.Model(&biz.Task{}).Where("status = ?", biz.TaskStatusRunning).Update("status", biz.TaskStatusFailed).Error; err != nil {
		r.log.Error("failed to clear zombie tasks", slog.Any("err", err))
	}
}

// processNext 取一条 waiting 任务执行，返回是否有任务被处理
func (r *Runner) processNext(ctx context.Context) bool {
	task := new(biz.Task)
	if err := r.db.Where("status = ?", biz.TaskStatusWaiting).Order("id asc").First(task).Error; err != nil {
		return false
	}

	r.execute(ctx, task)
	return true
}

// execute 执行单个任务
func (r *Runner) execute(ctx context.Context, task *biz.Task) {
	// 原子抢占，任务可能在取出后被取消
	result := r.db.Model(task).Where("status = ?", biz.TaskStatusWaiting).Update("status", biz.TaskStatusRunning)
	if result.Error != nil {
		r.log.Error("failed to update task status to running", slog.Any("task_id", task.ID), slog.Any("err", result.Error))
		return
	}
	if result.RowsAffected == 0 {
		return
	}

	// 计算日志路径并保存
	logDir := filepath.Join(app.Root, "panel/storage/logs/task")
	_ = os.MkdirAll(logDir, 0o700)
	logFile := filepath.Join(logDir, fmt.Sprintf("%d.log", task.ID))
	if err := r.db.Model(task).Update("log", logFile).Error; err != nil {
		r.log.Error("failed to update task log path", slog.Any("task_id", task.ID), slog.Any("err", err))
		return
	}

	// 登记当前任务，供 Cancel 定位
	taskCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.currentID, r.currentCancel = task.ID, cancel
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.currentID, r.currentCancel = 0, nil
		r.mu.Unlock()
		cancel()
	}()

	if err := shell.ExecWithLog(taskCtx, task.Shell, logFile); err != nil {
		// 用户取消标记为 canceled，面板停机保持 failed 由下次启动清理语义兜底
		status := biz.TaskStatusFailed
		if taskCtx.Err() != nil && ctx.Err() == nil {
			status = biz.TaskStatusCanceled
			r.runCancelShell(task, logFile)
		}
		r.log.Warn("background task did not finish", slog.Any("task_id", task.ID), slog.Any("status", status), slog.Any("err", err))
		_ = r.db.Model(task).Update("status", status).Error
		return
	}

	if err := r.db.Model(task).Update("status", biz.TaskStatusSuccess).Error; err != nil {
		r.log.Error("failed to update task status to success", slog.Any("task_id", task.ID), slog.Any("err", err))
	}
}

// runCancelShell 任务被取消后执行清理命令，输出追加到任务日志
func (r *Runner) runCancelShell(task *biz.Task, logFile string) {
	if task.CancelShell == "" {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		r.log.Warn("failed to open task log for cancel shell", slog.Any("task_id", task.ID), slog.Any("err", err))
		return
	}
	defer func(f *os.File) { _ = f.Close() }(f)

	cmd := exec.CommandContext(cleanupCtx, "bash", "-c", task.CancelShell)
	cmd.Stdout = f
	cmd.Stderr = f
	if err = cmd.Run(); err != nil {
		r.log.Warn("failed to run task cancel shell", slog.Any("task_id", task.ID), slog.Any("err", err))
	}
}
