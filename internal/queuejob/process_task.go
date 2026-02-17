package queuejob

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/shell"
)

// ProcessTask 处理面板任务
type ProcessTask struct {
	log      *slog.Logger
	taskRepo biz.TaskRepo
	taskID   uint
}

// NewProcessTask 实例化 ProcessTask
func NewProcessTask(log *slog.Logger, taskRepo biz.TaskRepo) *ProcessTask {
	return &ProcessTask{
		log:      log,
		taskRepo: taskRepo,
	}
}

func (r *ProcessTask) Handle(args ...any) error {
	taskID, ok := args[0].(uint)
	if !ok {
		return errors.New("参数错误")
	}
	r.taskID = taskID

	task, err := r.taskRepo.Get(taskID)
	if err != nil {
		return err
	}

	if err = r.taskRepo.UpdateStatus(taskID, biz.TaskStatusRunning); err != nil {
		return err
	}

	// 计算日志路径并保存
	logDir := filepath.Join(app.Root, "storage/logs/task")
	_ = os.MkdirAll(logDir, 0o700)
	logFile := filepath.Join(logDir, fmt.Sprintf("%d.log", taskID))
	if err = r.taskRepo.UpdateLog(taskID, logFile); err != nil {
		return err
	}

	if err = shell.ExecWithLog(task.Shell, logFile); err != nil {
		return err
	}

	if err = r.taskRepo.UpdateStatus(taskID, biz.TaskStatusSuccess); err != nil {
		return err
	}

	return nil
}

func (r *ProcessTask) ErrHandle(err error) {
	r.log.Warn("background task failed", slog.Any("task_id", r.taskID), slog.Any("err", err))
	_ = r.taskRepo.UpdateStatus(r.taskID, biz.TaskStatusFailed)
}
