package biz

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/utils/crypt"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
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
	Create(server *DatabaseServer) error
	Save(server *DatabaseServer) error
	UpdateRemark(req *request.DatabaseServerUpdateRemark) error
	UpdatePassword(name string, password string) error
	UpdatePort(name string, port uint) error
	Delete(id uint) error
	ClearUsers(id uint) error
	ListUsers(serverID uint) ([]*DatabaseUser, error)
	CreateUser(user *DatabaseUser) error
	Operator(server *DatabaseServer) (db.Operator, error)
	CheckServer(server *DatabaseServer) bool
}

// DatabaseServerUsecase 数据库服务器业务用例
type DatabaseServerUsecase struct {
	repo DatabaseServerRepo
	t    *gotext.Locale
	log  *slog.Logger
}

func NewDatabaseServerUsecase(i do.Injector) (*DatabaseServerUsecase, error) {
	return &DatabaseServerUsecase{
		repo: do.MustInvoke[DatabaseServerRepo](i),
		t:    do.MustInvoke[*gotext.Locale](i),
		log:  do.MustInvoke[*slog.Logger](i),
	}, nil
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
	databaseServer := &DatabaseServer{
		Name:     req.Name,
		Type:     DatabaseType(req.Type),
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Remark:   req.Remark,
	}

	if !uc.repo.CheckServer(databaseServer) {
		return errors.New(uc.t.Get("check server connection failed"))
	}

	return uc.repo.Create(databaseServer)
}

func (uc *DatabaseServerUsecase) Update(req *request.DatabaseServerUpdate) error {
	server, err := uc.repo.Get(req.ID)
	if err != nil {
		return err
	}

	server.Name = req.Name
	server.Host = req.Host
	server.Port = req.Port
	server.Username = req.Username
	server.Password = req.Password
	server.Remark = req.Remark

	if !uc.repo.CheckServer(server) {
		return errors.New(uc.t.Get("check server connection failed"))
	}

	return uc.repo.Save(server)
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
	server, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	// 非 Operator 类型不支持用户同步
	switch server.Type {
	case DatabaseTypeRedis, DatabaseTypeMongoDB, DatabaseTypeSQLite, DatabaseTypeElasticsearch:
		return fmt.Errorf("sync is not supported for %s", server.Type)
	}

	users, err := uc.repo.ListUsers(id)
	if err != nil {
		return err
	}

	operator, err := uc.repo.Operator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	switch server.Type {
	case DatabaseTypeMysql:
		allUsers, err := operator.Users()
		if err != nil {
			return err
		}
		for user := range slices.Values(allUsers) {
			if !slices.ContainsFunc(users, func(a *DatabaseUser) bool {
				return a.Username == user.User && a.Host == user.Host
			}) && !slices.Contains([]string{"root", "mysql.sys", "mysql.session", "mysql.infoschema"}, user.User) {
				newUser := &DatabaseUser{
					ServerID: id,
					Username: user.User,
					Host:     user.Host,
					Remark:   uc.t.Get("sync from server %s", server.Name),
				}
				if err = uc.repo.CreateUser(newUser); err != nil {
					uc.log.Warn("sync mysql database user failed", slog.String("type", OperationTypeDatabaseServer), slog.Uint64("operator_id", 0), slog.Any("err", err))
				}
			}
		}
	case DatabaseTypePostgresql:
		allUsers, err := operator.Users()
		if err != nil {
			return err
		}
		for user := range slices.Values(allUsers) {
			if !slices.ContainsFunc(users, func(a *DatabaseUser) bool {
				return a.Username == user.User
			}) && !slices.Contains([]string{"postgres"}, user.User) {
				newUser := &DatabaseUser{
					ServerID: id,
					Username: user.User,
					Remark:   uc.t.Get("sync from server %s", server.Name),
				}
				if err = uc.repo.CreateUser(newUser); err != nil {
					uc.log.Warn("sync postgresql database user failed", slog.String("type", OperationTypeDatabaseServer), slog.Uint64("operator_id", 0), slog.Any("err", err))
				}
			}
		}
	case DatabaseTypeClickHouse:
		allUsers, err := operator.Users()
		if err != nil {
			return err
		}
		for user := range slices.Values(allUsers) {
			if !slices.ContainsFunc(users, func(a *DatabaseUser) bool {
				return a.Username == user.User
			}) && !slices.Contains([]string{"default"}, user.User) {
				newUser := &DatabaseUser{
					ServerID: id,
					Username: user.User,
					Remark:   uc.t.Get("sync from server %s", server.Name),
				}
				if err = uc.repo.CreateUser(newUser); err != nil {
					uc.log.Warn("sync clickhouse database user failed", slog.String("type", OperationTypeDatabaseServer), slog.Uint64("operator_id", 0), slog.Any("err", err))
				}
			}
		}
	}

	return nil
}
