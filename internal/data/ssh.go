package data

import (
	"errors"
	"fmt"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	cryptossh "golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	pkgssh "github.com/acepanel/panel/v3/pkg/ssh"
)

type sshRepo struct {
	t  *gotext.Locale
	db *gorm.DB
}

func NewSSHRepo(i do.Injector) (biz.SSHRepo, error) {
	return &sshRepo{
		t:  do.MustInvoke[*gotext.Locale](i),
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

func (r *sshRepo) List(page, limit uint) ([]*biz.SSH, int64, error) {
	ssh := make([]*biz.SSH, 0)
	var total int64
	err := r.db.Model(&biz.SSH{}).Omit("Hosts").Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&ssh).Error
	return ssh, total, err
}

func (r *sshRepo) Get(id uint) (*biz.SSH, error) {
	ssh := new(biz.SSH)
	if err := r.db.Where("id = ?", id).First(ssh).Error; err != nil {
		return nil, err
	}

	return ssh, nil
}

func (r *sshRepo) Create(req *request.SSHCreate) error {
	conf := pkgssh.ClientConfig{
		AuthMethod: pkgssh.AuthMethod(req.AuthMethod),
		Host:       fmt.Sprintf("%s:%d", req.Host, req.Port),
		User:       req.User,
		Password:   req.Password,
		Key:        req.Key,
		Passphrase: req.Passphrase,
	}
	client, err := pkgssh.NewSSHClient(conf)
	if err != nil {
		return errors.New(r.t.Get("failed to check ssh connection: %v", err))
	}
	defer func(client *cryptossh.Client) { _ = client.Close() }(client)

	ssh := &biz.SSH{
		Name:   req.Name,
		Host:   req.Host,
		Port:   req.Port,
		Config: conf,
		Remark: req.Remark,
	}

	if err = r.db.Create(ssh).Error; err != nil {
		return err
	}

	return nil
}

func (r *sshRepo) Update(req *request.SSHUpdate) error {
	conf := pkgssh.ClientConfig{
		AuthMethod: pkgssh.AuthMethod(req.AuthMethod),
		Host:       fmt.Sprintf("%s:%d", req.Host, req.Port),
		User:       req.User,
		Password:   req.Password,
		Key:        req.Key,
		Passphrase: req.Passphrase,
	}
	client, err := pkgssh.NewSSHClient(conf)
	if err != nil {
		return errors.New(r.t.Get("failed to check ssh connection: %v", err))
	}
	defer func(client *cryptossh.Client) { _ = client.Close() }(client)

	ssh := &biz.SSH{
		ID:     req.ID,
		Name:   req.Name,
		Host:   req.Host,
		Port:   req.Port,
		Config: conf,
		Remark: req.Remark,
	}

	if err = r.db.Model(ssh).Where("id = ?", req.ID).Select("*").Updates(ssh).Error; err != nil {
		return err
	}

	return nil
}

func (r *sshRepo) Delete(id uint) error {
	if err := r.db.Delete(&biz.SSH{}, id).Error; err != nil {
		return err
	}

	return nil
}
