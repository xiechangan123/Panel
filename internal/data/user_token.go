package data

import (
	"time"

	"github.com/go-rat/utils/hash"
	"github.com/go-rat/utils/str"
	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/tnb-labs/panel/internal/biz"
)

type userTokenRepo struct {
	t      *gotext.Locale
	db     *gorm.DB
	hasher hash.Hasher
}

func NewUserTokenRepo(t *gotext.Locale, db *gorm.DB) biz.UserTokenRepo {
	return &userTokenRepo{
		t:      t,
		db:     db,
		hasher: hash.NewArgon2id(),
	}
}

func (r userTokenRepo) List(userID, page, limit uint) ([]*biz.UserToken, int64, error) {
	userTokens := make([]*biz.UserToken, 0)
	var total int64
	err := r.db.Model(&biz.UserToken{}).Where("user_id = ?", userID).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&userTokens).Error
	return userTokens, total, err
}

func (r userTokenRepo) Create(userID uint, ips []string, expired time.Time) (*biz.UserToken, error) {
	token := str.Random(32)
	hashedToken, err := r.hasher.Make(token)
	if err != nil {
		return nil, err
	}

	userToken := &biz.UserToken{
		UserID:    userID,
		Token:     hashedToken,
		IPs:       ips,
		ExpiredAt: expired,
	}
	if err = r.db.Create(userToken).Error; err != nil {
		return nil, err
	}

	userToken.Token = token

	return userToken, nil
}

func (r userTokenRepo) Get(id uint) (*biz.UserToken, error) {
	userToken := new(biz.UserToken)
	if err := r.db.First(userToken, id).Error; err != nil {
		return nil, err
	}

	return userToken, nil
}

func (r userTokenRepo) Delete(id uint) error {
	userToken := new(biz.UserToken)
	if err := r.db.First(userToken, id).Error; err != nil {
		return err
	}

	return r.db.Delete(userToken).Error
}

func (r userTokenRepo) Update(id uint, ips []string, expired time.Time) (*biz.UserToken, error) {
	userToken := new(biz.UserToken)
	if err := r.db.First(userToken, id).Error; err != nil {
		return nil, err
	}

	userToken.IPs = ips
	userToken.ExpiredAt = expired

	if err := r.db.Save(userToken).Error; err != nil {
		return nil, err
	}

	return userToken, nil
}
