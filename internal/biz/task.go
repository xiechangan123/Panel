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
