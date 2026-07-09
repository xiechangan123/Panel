package biz

import (
	"context"
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
	Create(ctx context.Context, username, password, email string) (*User, error)
	UpdateUsername(ctx context.Context, id uint, username string) error
	UpdatePassword(ctx context.Context, id uint, password string) error
	UpdateEmail(ctx context.Context, id uint, email string) error
	Delete(ctx context.Context, id uint) error
	CheckPassword(username, password string) (*User, error)
	IsTwoFA(username string) (bool, error)
	GenerateTwoFA(id uint) (image.Image, string, string, error)
	UpdateTwoFA(id uint, code, secret string) error
}

type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

func (uc *UserUsecase) List(page, limit uint) ([]*User, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *UserUsecase) Get(id uint) (*User, error) {
	return uc.repo.Get(id)
}

func (uc *UserUsecase) Create(ctx context.Context, username, password, email string) (*User, error) {
	return uc.repo.Create(ctx, username, password, email)
}

func (uc *UserUsecase) UpdateUsername(ctx context.Context, id uint, username string) error {
	return uc.repo.UpdateUsername(ctx, id, username)
}

func (uc *UserUsecase) UpdatePassword(ctx context.Context, id uint, password string) error {
	return uc.repo.UpdatePassword(ctx, id, password)
}

func (uc *UserUsecase) UpdateEmail(ctx context.Context, id uint, email string) error {
	return uc.repo.UpdateEmail(ctx, id, email)
}

func (uc *UserUsecase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *UserUsecase) CheckPassword(username, password string) (*User, error) {
	return uc.repo.CheckPassword(username, password)
}

func (uc *UserUsecase) IsTwoFA(username string) (bool, error) {
	return uc.repo.IsTwoFA(username)
}

func (uc *UserUsecase) GenerateTwoFA(id uint) (image.Image, string, string, error) {
	return uc.repo.GenerateTwoFA(id)
}

func (uc *UserUsecase) UpdateTwoFA(id uint, code, secret string) error {
	return uc.repo.UpdateTwoFA(id, code, secret)
}
