package biz

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"not null;default:'';unique" json:"username"`
	Password  string         `gorm:"not null;default:''" json:"password"`
	Email     string         `gorm:"not null;default:''" json:"email"`
	TwoFA     string         `gorm:"not null;default:''" json:"two_fa"` // 2FA secret，为空表示未开启
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type UserRepo interface {
	Create(username, password string) (*User, error)
	CheckPassword(username, password string) (*User, error)
	Get(id uint) (*User, error)
	Save(user *User) error
}
