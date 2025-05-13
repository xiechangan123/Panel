package biz

import (
	"time"
)

type UserToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Token     string    `gorm:"not null;default:'';unique" json:"token"`
	IPs       []string  `gorm:"not null;default:'[]';serializer:json" json:"ips"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserTokenRepo interface {
	Create(userID uint, ips []string) (*UserToken, error)
	Get(id uint) (*UserToken, error)
	Save(user *UserToken) error
}
