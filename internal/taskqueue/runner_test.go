package taskqueue

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

func newRunnerForTest(t *testing.T) *Runner {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	// file::memory: 每连接独立库,限单连接以让 runner goroutine 与主 goroutine 共享数据
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err = db.AutoMigrate(&biz.Task{}); err != nil {
		t.Fatal(err)
	}
	return NewRunner(db, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// 等待任务进入指定状态
func waitStatus(t *testing.T, db *gorm.DB, id uint, status biz.TaskStatus, timeout time.Duration) *biz.Task {
	t.Helper()
	deadline := time.Now().Add(timeout)
	task := new(biz.Task)
	for time.Now().Before(deadline) {
		if err := db.First(task, id).Error; err == nil && task.Status == status {
			return task
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("task %d did not reach status %s, current: %s", id, status, task.Status)
	return nil
}

func TestRunnerCancelRunning(t *testing.T) {
	r := newRunnerForTest(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "cleanup.done")

	task := &biz.Task{
		Name:        "sleep",
		Status:      biz.TaskStatusWaiting,
		Shell:       "sleep 60",
		CancelShell: "touch " + marker,
	}
	if err := r.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	r.Run(t.Context())

	// 等待任务进入运行状态后取消
	waitStatus(t, r.db, task.ID, biz.TaskStatusRunning, 3*time.Second)
	if !r.Cancel(task.ID) {
		t.Fatal("Cancel should hit the running task")
	}

	got := waitStatus(t, r.db, task.ID, biz.TaskStatusCanceled, 3*time.Second)
	if got.Log == "" {
		t.Fatal("task log path should be set")
	}

	// 取消清理命令应已执行
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("cancel shell was not executed")
}

func TestRunnerCancelMiss(t *testing.T) {
	r := newRunnerForTest(t)
	if r.Cancel(1) {
		t.Fatal("Cancel should miss when nothing is running")
	}
}

func TestRunnerWaitingCanceledNotExecuted(t *testing.T) {
	r := newRunnerForTest(t)

	// 已被标记取消的任务不应被运行器抢占执行
	task := &biz.Task{Name: "noop", Status: biz.TaskStatusCanceled, Shell: "true"}
	if err := r.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	r.Run(t.Context())

	time.Sleep(200 * time.Millisecond)
	got := new(biz.Task)
	if err := r.db.First(got, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != biz.TaskStatusCanceled {
		t.Fatalf("canceled task should stay canceled, got %s", got.Status)
	}
}
