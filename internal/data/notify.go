package data

import (
	"encoding/json"

	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

type notifyChannelRepo struct {
	db *gorm.DB
}

func NewNotifyChannelRepo(db *gorm.DB) (biz.NotifyChannelRepo, error) {
	return &notifyChannelRepo{
		db: db,
	}, nil
}

func (r *notifyChannelRepo) List(page, limit uint) ([]*biz.NotifyChannel, int64, error) {
	channels := make([]*biz.NotifyChannel, 0)
	var total int64
	err := r.db.Model(&biz.NotifyChannel{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&channels).Error
	return channels, total, err
}

func (r *notifyChannelRepo) All() ([]*biz.NotifyChannel, error) {
	channels := make([]*biz.NotifyChannel, 0)
	err := r.db.Order("id desc").Find(&channels).Error
	return channels, err
}

func (r *notifyChannelRepo) Get(id uint) (*biz.NotifyChannel, error) {
	channel := new(biz.NotifyChannel)
	if err := r.db.Where("id = ?", id).First(channel).Error; err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *notifyChannelRepo) GetByIDs(ids []uint) ([]*biz.NotifyChannel, error) {
	channels := make([]*biz.NotifyChannel, 0)
	if len(ids) == 0 {
		return channels, nil
	}
	err := r.db.Where("id IN ?", ids).Find(&channels).Error
	return channels, err
}

func (r *notifyChannelRepo) Create(channel *biz.NotifyChannel) error {
	return r.db.Create(channel).Error
}

func (r *notifyChannelRepo) Update(channel *biz.NotifyChannel) error {
	return r.db.Save(channel).Error
}

// Delete 删除渠道，同时清理告警规则与事件设置中的引用
// 残留引用不会报错，只会让通知静默失效，因此必须一并清除
func (r *notifyChannelRepo) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&biz.NotifyChannel{}).Error; err != nil {
			return err
		}

		rules := make([]*biz.AlertRule, 0)
		if err := tx.Find(&rules).Error; err != nil {
			return err
		}
		for _, rule := range rules {
			channels := lo.Without(rule.Channels, id)
			if len(channels) == len(rule.Channels) {
				continue
			}
			rule.Channels = channels
			if err := tx.Save(rule).Error; err != nil {
				return err
			}
		}

		return r.removeEventChannel(tx, id)
	})
}

// removeEventChannel 从系统事件通知设置中移除指定渠道
func (r *notifyChannelRepo) removeEventChannel(tx *gorm.DB, id uint) error {
	setting := new(biz.Setting)
	if err := tx.Where("key = ?", biz.SettingKeyNotifyEventChannels).First(setting).Error; err != nil {
		// 未配置过事件通知
		return nil
	}

	channels := make([]uint, 0)
	if json.Unmarshal([]byte(setting.Value), &channels) != nil {
		return nil
	}

	remain := lo.Without(channels, id)
	if len(remain) == len(channels) {
		return nil
	}

	value, err := json.Marshal(remain)
	if err != nil {
		return err
	}
	setting.Value = string(value)

	return tx.Save(setting).Error
}
