package biz

import (
	"context"
	"time"

	"github.com/libtnb/utils/crypt"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type BackupStorageType string

const (
	BackupStorageTypeLocal  BackupStorageType = "local"
	BackupStorageTypeS3     BackupStorageType = "s3"
	BackupStorageTypeSFTP   BackupStorageType = "sftp"
	BackupStorageTypeWebDAV BackupStorageType = "webdav"
)

type BackupStorage struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	Type      BackupStorageType       `gorm:"not null;default:''" json:"type"`
	Name      string                  `gorm:"not null;default:''" json:"name"`
	Info      types.BackupStorageInfo `gorm:"not null;default:'{}';serializer:json" json:"info"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

func (r *BackupStorage) BeforeSave(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	switch r.Type {
	case BackupStorageTypeS3:
		r.Info.AccessKey, err = crypter.Encrypt([]byte(r.Info.AccessKey))
		if err != nil {
			return err
		}
		r.Info.SecretKey, err = crypter.Encrypt([]byte(r.Info.SecretKey))
		if err != nil {
			return err
		}
		return nil
	case BackupStorageTypeSFTP:
		r.Info.Username, err = crypter.Encrypt([]byte(r.Info.Username))
		if err != nil {
			return err
		}
		if r.Info.Password != "" {
			r.Info.Password, err = crypter.Encrypt([]byte(r.Info.Password))
			if err != nil {
				return err
			}
		}
		if r.Info.PrivateKey != "" {
			r.Info.PrivateKey, err = crypter.Encrypt([]byte(r.Info.PrivateKey))
			if err != nil {
				return err
			}
		}
	case BackupStorageTypeWebDAV:
		r.Info.Username, err = crypter.Encrypt([]byte(r.Info.Username))
		if err != nil {
			return err
		}
		r.Info.Password, err = crypter.Encrypt([]byte(r.Info.Password))
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (r *BackupStorage) AfterFind(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	switch r.Type {
	case BackupStorageTypeS3:
		accessKey, err := crypter.Decrypt(r.Info.AccessKey)
		if err == nil {
			r.Info.AccessKey = string(accessKey)
		}
		secretKey, err := crypter.Decrypt(r.Info.SecretKey)
		if err == nil {
			r.Info.SecretKey = string(secretKey)
		}
		return nil
	case BackupStorageTypeSFTP:
		username, err := crypter.Decrypt(r.Info.Username)
		if err == nil {
			r.Info.Username = string(username)
		}
		if r.Info.Password != "" {
			password, err := crypter.Decrypt(r.Info.Password)
			if err == nil {
				r.Info.Password = string(password)
			}
		}
		if r.Info.PrivateKey != "" {
			privateKey, err := crypter.Decrypt(r.Info.PrivateKey)
			if err == nil {
				r.Info.PrivateKey = string(privateKey)
			}
		}
	case BackupStorageTypeWebDAV:
		username, err := crypter.Decrypt(r.Info.Username)
		if err == nil {
			r.Info.Username = string(username)
		}
		password, err := crypter.Decrypt(r.Info.Password)
		if err == nil {
			r.Info.Password = string(password)
		}
		return nil
	}

	return nil
}

type BackupAccountRepo interface {
	List(page, limit uint) ([]*BackupStorage, int64, error)
	Get(id uint) (*BackupStorage, error)
	Create(ctx context.Context, req *request.BackupStorageCreate) (*BackupStorage, error)
	Update(ctx context.Context, req *request.BackupStorageUpdate) error
	Delete(ctx context.Context, id uint) error
}
