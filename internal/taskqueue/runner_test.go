package taskqueue

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leonelquinteros/gotext"
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
	return NewRunner(db, slog.New(slog.NewTextHandler(os.Stderr, nil)), stubNotifier{}, gotext.NewLocale("", "en"))
}

type stubNotifier struct{}

func (stubNotifier) SendEvent(biz.NotifyEvent, string, string) {}

// 等待任务进入指定状态
func waitStatus(t *testing.T, db *gorm.DB, id uint, status biz.TaskStatus) *biz.Task {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
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
	marker := filepath.Join(t.TempDir(), "cleanup.done")

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
	waitStatus(t, r.db, task.ID, biz.TaskStatusRunning)
	if !r.Cancel(task.ID) {
		t.Fatal("Cancel should hit the running task")
	}

	// 清理命令在写 canceled 之前同步跑完，状态到位即可断言产物
	got := waitStatus(t, r.db, task.ID, biz.TaskStatusCanceled)
	if got.Log == "" {
		t.Fatal("task log path should be set")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("cancel shell was not executed")
	}
}

// 模拟 systemd 停机：命令被 SIGTERM 杀死，运行器 ctx 几乎同时取消，应记为 canceled 并跑清理
func TestRunnerShutdownCanceled(t *testing.T) {
	r := newRunnerForTest(t)
	marker := filepath.Join(t.TempDir(), "cleanup.done")

	task := &biz.Task{
		Name:        "shutdown",
		Status:      biz.TaskStatusWaiting,
		Shell:       "kill -TERM $$",
		CancelShell: "touch " + marker,
	}
	if err := r.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	r.Run(ctx)

	waitStatus(t, r.db, task.ID, biz.TaskStatusRunning)
	cancel()

	waitStatus(t, r.db, task.ID, biz.TaskStatusCanceled)
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("cancel shell was not executed")
	}
}

// 没有停机信号跟进时，被信号杀死的命令仍是失败
func TestRunnerSignaledFailed(t *testing.T) {
	r := newRunnerForTest(t)

	task := &biz.Task{Name: "signaled", Status: biz.TaskStatusWaiting, Shell: "kill -TERM $$"}
	if err := r.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}

	r.Run(t.Context())

	waitStatus(t, r.db, task.ID, biz.TaskStatusFailed)
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
