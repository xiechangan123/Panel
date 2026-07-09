package biz

import (
	"time"

	"github.com/libtnb/utils/crypt"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
)

type DatabaseServerStatus string

const (
	DatabaseServerStatusValid   DatabaseServerStatus = "valid"
	DatabaseServerStatusInvalid DatabaseServerStatus = "invalid"
)

type DatabaseServer struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	Name      string               `gorm:"not null;default:'';unique" json:"name"`
	Type      DatabaseType         `gorm:"not null;default:''" json:"type"`
	Host      string               `gorm:"not null;default:''" json:"host"`
	Port      uint                 `gorm:"not null;default:0" json:"port"`
	Username  string               `gorm:"not null;default:''" json:"username"`
	Password  string               `gorm:"not null;default:''" json:"password"`
	Status    DatabaseServerStatus `gorm:"-:all" json:"status"`
	Remark    string               `gorm:"not null;default:''" json:"remark"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

func (r *DatabaseServer) BeforeSave(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	r.Password, err = crypter.Encrypt([]byte(r.Password))
	if err != nil {
		return err
	}

	return nil

}

func (r *DatabaseServer) AfterFind(tx *gorm.DB) error {
	crypter, err := crypt.NewXChacha20Poly1305([]byte(app.Key))
	if err != nil {
		return err
	}

	password, err := crypter.Decrypt(r.Password)
	if err == nil {
		r.Password = string(password)
	}

	return nil
}

type DatabaseServerRepo interface {
	Count() (int64, error)
	List(page, limit uint, typ string) ([]*DatabaseServer, int64, error)
	Get(id uint) (*DatabaseServer, error)
	GetByName(name string) (*DatabaseServer, error)
	Create(req *request.DatabaseServerCreate) error
	Update(req *request.DatabaseServerUpdate) error
	UpdateRemark(req *request.DatabaseServerUpdateRemark) error
	UpdatePassword(name string, password string) error
	UpdatePort(name string, port uint) error
	Delete(id uint) error
	ClearUsers(id uint) error
	Sync(id uint) error
}

// DatabaseServerUsecase 数据库服务器业务用例
type DatabaseServerUsecase struct {
	repo DatabaseServerRepo
}

func NewDatabaseServerUsecase(repo DatabaseServerRepo) *DatabaseServerUsecase {
	return &DatabaseServerUsecase{repo: repo}
}

func (uc *DatabaseServerUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *DatabaseServerUsecase) List(page, limit uint, typ string) ([]*DatabaseServer, int64, error) {
	return uc.repo.List(page, limit, typ)
}

func (uc *DatabaseServerUsecase) Get(id uint) (*DatabaseServer, error) {
	return uc.repo.Get(id)
}

func (uc *DatabaseServerUsecase) GetByName(name string) (*DatabaseServer, error) {
	return uc.repo.GetByName(name)
}

func (uc *DatabaseServerUsecase) Create(req *request.DatabaseServerCreate) error {
	return uc.repo.Create(req)
}

func (uc *DatabaseServerUsecase) Update(req *request.DatabaseServerUpdate) error {
	return uc.repo.Update(req)
}

func (uc *DatabaseServerUsecase) UpdateRemark(req *request.DatabaseServerUpdateRemark) error {
	return uc.repo.UpdateRemark(req)
}

func (uc *DatabaseServerUsecase) UpdatePassword(name string, password string) error {
	return uc.repo.UpdatePassword(name, password)
}

func (uc *DatabaseServerUsecase) UpdatePort(name string, port uint) error {
	return uc.repo.UpdatePort(name, port)
}

func (uc *DatabaseServerUsecase) Delete(id uint) error {
	return uc.repo.Delete(id)
}

func (uc *DatabaseServerUsecase) ClearUsers(id uint) error {
	return uc.repo.ClearUsers(id)
}

func (uc *DatabaseServerUsecase) Sync(id uint) error {
	return uc.repo.Sync(id)
}
