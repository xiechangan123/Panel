package biz

import (
	"image"
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

	Tokens []*UserToken `gorm:"foreignKey:UserID" json:"-"`
}

type UserRepo interface {
	List(page, limit uint) ([]*User, int64, error)
	Get(id uint) (*User, error)
	Create(username, password, email string) (*User, error)
	UpdateUsername(id uint, username string) error
	UpdatePassword(id uint, password string) error
	UpdateEmail(id uint, email string) error
	Delete(id uint) error
	CheckPassword(username, password string) (*User, error)
	IsTwoFA(username string) (bool, error)
	GenerateTwoFA(id uint) (image.Image, string, string, error)
	UpdateTwoFA(id uint, code, secret string) error
}
