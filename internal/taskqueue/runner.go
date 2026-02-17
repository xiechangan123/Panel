package taskqueue

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/shell"
)

type Runner struct {
	db     *gorm.DB
	log    *slog.Logger
	notify chan struct{}
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
		if !r.processNext() {
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
func (r *Runner) processNext() bool {
	task := new(biz.Task)
	if err := r.db.Where("status = ?", biz.TaskStatusWaiting).Order("id asc").First(task).Error; err != nil {
		return false
	}

	r.execute(task)
	return true
}

// execute 执行单个任务
func (r *Runner) execute(task *biz.Task) {
	if err := r.db.Model(task).Update("status", biz.TaskStatusRunning).Error; err != nil {
		r.log.Error("failed to update task status to running", slog.Any("task_id", task.ID), slog.Any("err", err))
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

	if err := shell.ExecWithLog(task.Shell, logFile); err != nil {
		r.log.Warn("failed to execute background task", slog.Any("task_id", task.ID), slog.Any("err", err))
		_ = r.db.Model(task).Update("status", biz.TaskStatusFailed).Error
		return
	}

	if err := r.db.Model(task).Update("status", biz.TaskStatusSuccess).Error; err != nil {
		r.log.Error("failed to update task status to success", slog.Any("task_id", task.ID), slog.Any("err", err))
	}
}
