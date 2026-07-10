package biz

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/libtnb/utils/crypt"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type DatabaseUserStatus string

const (
	DatabaseUserStatusValid   DatabaseUserStatus = "valid"
	DatabaseUserStatusInvalid DatabaseUserStatus = "invalid"
)

type DatabaseUser struct {
	ID         uint               `gorm:"primaryKey" json:"id"`
	ServerID   uint               `gorm:"not null;default:0" json:"server_id"`
	Username   string             `gorm:"not null;default:''" json:"username"`
	Password   string             `gorm:"not null;default:''" json:"password"`
	Host       string             `gorm:"not null;default:''" json:"host"` // 仅 mysql
	Status     DatabaseUserStatus `gorm:"-:all" json:"status"`             // 仅显示
	Privileges []string           `gorm:"-:all" json:"privileges"`         // 仅显示
	Remark     string             `gorm:"not null;default:''" json:"remark"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`

	Server *DatabaseServer `gorm:"foreignKey:ServerID;references:ID" json:"server"`
}

func (r *DatabaseUser) BeforeSave(tx *gorm.DB) error {
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

func (r *DatabaseUser) AfterFind(tx *gorm.DB) error {
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

type DatabaseUserRepo interface {
	Count() (int64, error)
	List(page, limit uint, typ string) ([]*DatabaseUser, int64, error)
	Get(id uint) (*DatabaseUser, error)
	UpdateRemark(req *request.DatabaseUserUpdateRemark) error
	Operator(server *DatabaseServer) (db.Operator, error)
	Upsert(user *DatabaseUser) error
	Save(user *DatabaseUser) error
	DeleteByID(id uint) error
	ListByNames(serverID uint, names []string) ([]*DatabaseUser, error)
	DeleteByServerNames(serverID uint, names []string) error
}

// DatabaseUserUsecase 数据库用户业务用例
type DatabaseUserUsecase struct {
	repo   DatabaseUserRepo
	server DatabaseServerRepo
	log    *slog.Logger
}

func NewDatabaseUserUsecase(i do.Injector) (*DatabaseUserUsecase, error) {
	return &DatabaseUserUsecase{
		repo:   do.MustInvoke[DatabaseUserRepo](i),
		server: do.MustInvoke[DatabaseServerRepo](i),
		log:    do.MustInvoke[*slog.Logger](i),
	}, nil
}

func (uc *DatabaseUserUsecase) Count() (int64, error) {
	return uc.repo.Count()
}

func (uc *DatabaseUserUsecase) List(page, limit uint, typ string) ([]*DatabaseUser, int64, error) {
	return uc.repo.List(page, limit, typ)
}

func (uc *DatabaseUserUsecase) Get(id uint) (*DatabaseUser, error) {
	return uc.repo.Get(id)
}

func (uc *DatabaseUserUsecase) Create(ctx context.Context, req *request.DatabaseUserCreate) error {
	server, err := uc.server.Get(req.ServerID)
	if err != nil {
		return err
	}

	operator, err := uc.repo.Operator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	// 创建用户
	if err = operator.UserCreate(req.Username, req.Password, req.Host); err != nil {
		return err
	}

	// 创建数据库并授权
	for name := range slices.Values(req.Privileges) {
		if err = operator.DatabaseCreate(name); err != nil {
			return err
		}
		if err = operator.PrivilegesGrant(req.Username, name, req.Host); err != nil {
			return err
		}
	}

	user := &DatabaseUser{
		ServerID: req.ServerID,
		Username: req.Username,
		Host:     req.Host,
		Password: req.Password,
		Remark:   req.Remark,
	}

	if err = uc.repo.Upsert(user); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("database user created", slog.String("type", OperationTypeDatabaseUser), slog.Uint64("operator_id", operatorID(ctx)), slog.String("username", req.Username), slog.Uint64("server_id", uint64(req.ServerID)))

	return nil
}

func (uc *DatabaseUserUsecase) Update(req *request.DatabaseUserUpdate) error {
	user, err := uc.repo.Get(req.ID)
	if err != nil {
		return err
	}

	server, err := uc.server.Get(user.ServerID)
	if err != nil {
		return err
	}

	operator, err := uc.repo.Operator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	// 更新密码
	if req.Password != "" {
		if err = operator.UserPassword(user.Username, req.Password, user.Host); err != nil {
			return err
		}
		user.Password = req.Password
	}

	// 撤销被移除的权限
	currentPrivileges, _ := operator.UserPrivileges(user.Username, user.Host)
	for name := range slices.Values(currentPrivileges) {
		if !slices.Contains(req.Privileges, name) {
			_ = operator.PrivilegesRevoke(user.Username, name, user.Host)
		}
	}

	// 创建数据库并授权
	for name := range slices.Values(req.Privileges) {
		if err = operator.DatabaseCreate(name); err != nil {
			return err
		}
		if err = operator.PrivilegesGrant(user.Username, name, user.Host); err != nil {
			return err
		}
	}

	user.Remark = req.Remark

	return uc.repo.Save(user)
}

func (uc *DatabaseUserUsecase) UpdateRemark(req *request.DatabaseUserUpdateRemark) error {
	return uc.repo.UpdateRemark(req)
}

func (uc *DatabaseUserUsecase) Delete(ctx context.Context, id uint) error {
	user, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	server, err := uc.server.Get(user.ServerID)
	if err != nil {
		return err
	}

	operator, err := uc.repo.Operator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	_ = operator.UserDrop(user.Username, user.Host)

	if err = uc.repo.DeleteByID(id); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("database user deleted", slog.String("type", OperationTypeDatabaseUser), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("username", user.Username))

	return nil
}

func (uc *DatabaseUserUsecase) DeleteByNames(serverID uint, names []string) error {
	server, err := uc.server.Get(serverID)
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
		users, err := uc.repo.ListByNames(serverID, names)
		if err != nil {
			return err
		}
		for name := range slices.Values(names) {
			host := "localhost"
			for u := range slices.Values(users) {
				if u.Username == name {
					host = u.Host
					break
				}
			}
			_ = operator.UserDrop(name, host)
		}
	case DatabaseTypePostgresql, DatabaseTypeClickHouse:
		for name := range slices.Values(names) {
			_ = operator.UserDrop(name)
		}
	}

	return uc.repo.DeleteByServerNames(serverID, names)
}
