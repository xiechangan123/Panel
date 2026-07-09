package biz

import (
	"time"

	"gorm.io/gorm"
)

type UserPasskey struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	Name           string         `gorm:"not null;default:''" json:"name"`
	CredentialID   []byte         `gorm:"uniqueIndex;not null" json:"-"`
	PublicKey      []byte         `gorm:"not null" json:"-"`
	AAGUID         []byte         `json:"-"`
	SignCount      uint32         `gorm:"not null;default:0" json:"-"`
	Transports     string         `gorm:"not null;default:''" json:"transports"` // JSON: ["internal","hybrid"]
	BackupEligible bool           `gorm:"not null;default:false" json:"-"`
	BackupState    bool           `gorm:"not null;default:false" json:"-"`
	LastUsedAt     *time.Time     `json:"last_used_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type UserPasskeyRepo interface {
	List(userID uint) ([]*UserPasskey, error)
	Create(passkey *UserPasskey) error
	UpdateSignCount(credentialID []byte, signCount uint32) error
	UpdateLastUsed(credentialID []byte) error
	Delete(userID, id uint) error
	GetByCredentialID(credentialID []byte) (*UserPasskey, *User, error)
	HasPasskey(userID uint) (bool, error)
	HasAny() (bool, error)
	DeleteAllByUserID(userID uint) error
}

type UserPasskeyUsecase struct {
	repo UserPasskeyRepo
}

func NewUserPasskeyUsecase(repo UserPasskeyRepo) *UserPasskeyUsecase {
	return &UserPasskeyUsecase{repo: repo}
}

func (uc *UserPasskeyUsecase) List(userID uint) ([]*UserPasskey, error) {
	return uc.repo.List(userID)
}

func (uc *UserPasskeyUsecase) Create(passkey *UserPasskey) error {
	return uc.repo.Create(passkey)
}

func (uc *UserPasskeyUsecase) UpdateSignCount(credentialID []byte, signCount uint32) error {
	return uc.repo.UpdateSignCount(credentialID, signCount)
}

func (uc *UserPasskeyUsecase) UpdateLastUsed(credentialID []byte) error {
	return uc.repo.UpdateLastUsed(credentialID)
}

func (uc *UserPasskeyUsecase) Delete(userID, id uint) error {
	return uc.repo.Delete(userID, id)
}

func (uc *UserPasskeyUsecase) GetByCredentialID(credentialID []byte) (*UserPasskey, *User, error) {
	return uc.repo.GetByCredentialID(credentialID)
}

func (uc *UserPasskeyUsecase) HasPasskey(userID uint) (bool, error) {
	return uc.repo.HasPasskey(userID)
}

func (uc *UserPasskeyUsecase) HasAny() (bool, error) {
	return uc.repo.HasAny()
}

func (uc *UserPasskeyUsecase) DeleteAllByUserID(userID uint) error {
	return uc.repo.DeleteAllByUserID(userID)
}
