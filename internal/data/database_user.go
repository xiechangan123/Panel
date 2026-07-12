package data

import (
	"fmt"
	"slices"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type databaseUserRepo struct {
	db *gorm.DB
}

func NewDatabaseUserRepo(i do.Injector) (biz.DatabaseUserRepo, error) {
	return &databaseUserRepo{
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

func (r *databaseUserRepo) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&biz.DatabaseUser{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *databaseUserRepo) List(page, limit uint, typ string) ([]*biz.DatabaseUser, int64, error) {
	user := make([]*biz.DatabaseUser, 0)
	var total int64
	query := r.db.Model(&biz.DatabaseUser{}).Preload("Server").Order("id desc")
	if typ != "" {
		query = query.Joins("JOIN database_servers ON database_servers.id = database_users.server_id").Where("database_servers.type = ?", typ)
	}
	err := query.Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&user).Error

	for u := range slices.Values(user) {
		r.fillUser(u)
	}

	return user, total, err
}

func (r *databaseUserRepo) Get(id uint) (*biz.DatabaseUser, error) {
	user := new(biz.DatabaseUser)
	if err := r.db.Preload("Server").Where("id = ?", id).First(user).Error; err != nil {
		return nil, err
	}

	r.fillUser(user)

	return user, nil
}

func (r *databaseUserRepo) UpdateRemark(req *request.DatabaseUserUpdateRemark) error {
	user, err := r.Get(req.ID)
	if err != nil {
		return err
	}

	user.Remark = req.Remark

	return r.db.Save(user).Error
}

// Operator 获取数据库操作句柄
func (r *databaseUserRepo) Operator(server *biz.DatabaseServer) (db.Operator, error) {
	switch server.Type {
	case biz.DatabaseTypeMysql:
		return newMySQLOperator(server.Username, server.Password, server.Host, server.Port)
	case biz.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return nil, err
		}
		return postgres, nil
	case biz.DatabaseTypeClickHouse:
		clickhouse, err := db.NewClickHouse(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return nil, err
		}
		return clickhouse, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", server.Type)
	}
}

// Upsert 创建或更新用户记录（FirstOrInit + Save，非原子保留）
func (r *databaseUserRepo) Upsert(user *biz.DatabaseUser) error {
	if err := r.db.FirstOrInit(user, user).Error; err != nil {
		return err
	}

	return r.db.Save(user).Error
}

// Save 保存用户记录
func (r *databaseUserRepo) Save(user *biz.DatabaseUser) error {
	return r.db.Save(user).Error
}

// DeleteByID 按 ID 删除用户记录
func (r *databaseUserRepo) DeleteByID(id uint) error {
	return r.db.Where("id = ?", id).Delete(&biz.DatabaseUser{}).Error
}

// ListByNames 查询指定服务器下匹配用户名的用户记录
func (r *databaseUserRepo) ListByNames(serverID uint, names []string) ([]*biz.DatabaseUser, error) {
	users := make([]*biz.DatabaseUser, 0)
	if err := r.db.Where("server_id = ? AND username IN ?", serverID, names).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// DeleteByServerNames 按服务器与用户名删除用户记录
func (r *databaseUserRepo) DeleteByServerNames(serverID uint, names []string) error {
	return r.db.Where("server_id = ? AND username IN ?", serverID, names).Delete(&biz.DatabaseUser{}).Error
}

func (r *databaseUserRepo) fillUser(user *biz.DatabaseUser) {
	server, err := r.loadServer(user.ServerID)
	if err == nil {
		operator, err := r.Operator(server)
		if err == nil {
			defer operator.Close()
			switch server.Type {
			case biz.DatabaseTypeMysql:
				privileges, _ := operator.UserPrivileges(user.Username, user.Host)
				user.Privileges = privileges
				if mysql2, err := newMySQLOperator(user.Username, user.Password, server.Host, server.Port); err == nil {
					mysql2.Close()
					user.Status = biz.DatabaseUserStatusValid
				} else {
					user.Status = biz.DatabaseUserStatusInvalid
				}
			case biz.DatabaseTypePostgresql:
				privileges, _ := operator.UserPrivileges(user.Username)
				user.Privileges = privileges
				if postgres2, err := db.NewPostgres(user.Username, user.Password, server.Host, server.Port); err == nil {
					postgres2.Close()
					user.Status = biz.DatabaseUserStatusValid
				} else {
					user.Status = biz.DatabaseUserStatusInvalid
				}
			case biz.DatabaseTypeClickHouse:
				privileges, _ := operator.UserPrivileges(user.Username)
				user.Privileges = privileges
				if ch2, err := db.NewClickHouse(user.Username, user.Password, fmt.Sprintf("%s:%d", server.Host, server.Port)); err == nil {
					ch2.Close()
					user.Status = biz.DatabaseUserStatusValid
				} else {
					user.Status = biz.DatabaseUserStatusInvalid
				}
			}
		} else {
			user.Status = biz.DatabaseUserStatusInvalid
		}
	}
	// 初始化，防止 nil
	if user.Privileges == nil {
		user.Privileges = make([]string, 0)
	}
}

// loadServer 直读服务器记录，断开 repo→repo，仅取连接信息
func (r *databaseUserRepo) loadServer(id uint) (*biz.DatabaseServer, error) {
	server := new(biz.DatabaseServer)
	if err := r.db.Where("id = ?", id).First(server).Error; err != nil {
		return nil, err
	}

	return server, nil
}
