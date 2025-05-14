package biz

import (
	"time"

	"github.com/go-rat/utils/hash"
	"gorm.io/gorm"
)

type UserToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Token     string    `gorm:"not null;default:'';unique" json:"-"`
	IPs       []string  `gorm:"not null;default:'[]';serializer:json" json:"ips"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *UserToken) BeforeSave(tx *gorm.DB) error {
	hasher := hash.NewArgon2id()
	var err error

	r.Token, err = hasher.Make(r.Token)
	if err != nil {
		return err
	}

	return nil
}

type UserTokenRepo interface {
	List(userID, page, limit uint) ([]*UserToken, int64, error)
	Create(userID uint, ips []string, expired time.Time) (*UserToken, error)
	Get(id uint) (*UserToken, error)
	Delete(id uint) error
	Update(id uint, ips []string, expired time.Time) (*UserToken, error)
}
