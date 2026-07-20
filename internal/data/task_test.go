package data

import (
	"context"
	"testing"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

type stubRunner struct{}

func (stubRunner) Run(context.Context) {}
func (stubRunner) Notify()             {}
func (stubRunner) Cancel(uint) bool    { return false }

func newTaskRepoForTest(t *testing.T) *taskRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&biz.Task{}); err != nil {
		t.Fatal(err)
	}
	return &taskRepo{t: gotext.NewLocale("", "en"), db: db, runner: stubRunner{}}
}

func TestTaskPushDedup(t *testing.T) {
	repo := newTaskRepoForTest(t)

	// shell 含随机片段也按 key 去重
	if err := repo.Push(&biz.Task{Key: "backup:website:a", Name: "备份", Status: biz.TaskStatusWaiting, Shell: "echo 1"}); err != nil {
		t.Fatalf("first Push: %v", err)
	}
	if err := repo.Push(&biz.Task{Key: "backup:website:a", Name: "Backup", Status: biz.TaskStatusWaiting, Shell: "echo 2"}); err == nil {
		t.Fatal("duplicate key should be rejected")
	}

	// 不同 key 放行
	if err := repo.Push(&biz.Task{Key: "backup:website:b", Name: "Backup", Status: biz.TaskStatusWaiting, Shell: "echo 3"}); err != nil {
		t.Fatalf("different key should pass: %v", err)
	}

	// 终态后同 key 放行
	if err := repo.db.Model(&biz.Task{}).Where("`key` = ?", "backup:website:a").Update("status", biz.TaskStatusCanceled).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Push(&biz.Task{Key: "backup:website:a", Name: "Backup", Status: biz.TaskStatusWaiting, Shell: "echo 4"}); err != nil {
		t.Fatalf("Push after terminal status should pass: %v", err)
	}
}
