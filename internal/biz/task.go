package biz

import "time"

type TaskStatus string

const (
	TaskStatusWaiting TaskStatus = "waiting"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "finished"
	TaskStatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null;default:'';index" json:"name"`
	Status    TaskStatus `gorm:"not null;default:'waiting'" json:"status"`
	Shell     string     `gorm:"not null;default:''" json:"-"`
	Log       string     `gorm:"not null;default:''" json:"log"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type TaskRepo interface {
	HasRunningTask() bool
	List(page, limit uint) ([]*Task, int64, error)
	Get(id uint) (*Task, error)
	Delete(id uint) error
	UpdateStatus(id uint, status TaskStatus) error
	UpdateLog(id uint, log string) error
	Push(task *Task) error
}

type TaskUsecase struct {
	repo TaskRepo
}

func NewTaskUsecase(repo TaskRepo) *TaskUsecase {
	return &TaskUsecase{repo: repo}
}

func (uc *TaskUsecase) HasRunningTask() bool {
	return uc.repo.HasRunningTask()
}

func (uc *TaskUsecase) List(page, limit uint) ([]*Task, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *TaskUsecase) Get(id uint) (*Task, error) {
	return uc.repo.Get(id)
}

func (uc *TaskUsecase) Delete(id uint) error {
	return uc.repo.Delete(id)
}

func (uc *TaskUsecase) UpdateStatus(id uint, status TaskStatus) error {
	return uc.repo.UpdateStatus(id, status)
}

func (uc *TaskUsecase) UpdateLog(id uint, log string) error {
	return uc.repo.UpdateLog(id, log)
}

func (uc *TaskUsecase) Push(task *Task) error {
	return uc.repo.Push(task)
}
