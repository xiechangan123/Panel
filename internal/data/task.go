package data

import (
	"errors"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/types"
)

type taskRepo struct {
	t      *gotext.Locale
	db     *gorm.DB
	log    *slog.Logger
	runner types.TaskRunner
}

func NewTaskRepo(i do.Injector) (biz.TaskRepo, error) {
	return &taskRepo{
		t:      do.MustInvoke[*gotext.Locale](i),
		db:     do.MustInvoke[*gorm.DB](i),
		log:    do.MustInvoke[*slog.Logger](i),
		runner: do.MustInvoke[types.TaskRunner](i),
	}, nil
}

func (r *taskRepo) HasRunningTask() bool {
	var count int64
	r.db.Model(&biz.Task{}).Where("status = ?", biz.TaskStatusRunning).Or("status = ?", biz.TaskStatusWaiting).Count(&count)
	return count > 0
}

func (r *taskRepo) List(page, limit uint) ([]*biz.Task, int64, error) {
	tasks := make([]*biz.Task, 0)
	var total int64
	err := r.db.Model(&biz.Task{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&tasks).Error
	return tasks, total, err
}

func (r *taskRepo) Get(id uint) (*biz.Task, error) {
	task := new(biz.Task)
	err := r.db.Model(&biz.Task{}).Where("id = ?", id).First(task).Error
	return task, err
}

func (r *taskRepo) Delete(id uint) error {
	return r.db.Model(&biz.Task{}).Where("id = ?", id).Delete(&biz.Task{}).Error
}

func (r *taskRepo) UpdateStatus(id uint, status biz.TaskStatus) error {
	return r.db.Model(&biz.Task{}).Where("id = ?", id).Update("status", status).Error
}

func (r *taskRepo) UpdateLog(id uint, log string) error {
	return r.db.Model(&biz.Task{}).Where("id = ?", id).Update("log", log).Error
}

func (r *taskRepo) Push(task *biz.Task) error {
	// 防止有人喜欢酒吧点炒饭
	var count int64
	if err := r.db.Model(&biz.Task{}).Where("shell = ? and (status = ? or status = ?)", task.Shell, biz.TaskStatusWaiting, biz.TaskStatusRunning).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(r.t.Get("duplicate submission, please wait for the previous task to end"))
	}

	if err := r.db.Create(task).Error; err != nil {
		return err
	}

	r.runner.Notify()
	return nil
}
