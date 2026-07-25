package data

import (
	"encoding/json"
	"testing"

	"github.com/libtnb/sqlite"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

func newDBForTest(t *testing.T) *gorm.DB {
	t.Helper()
	// 渠道配置落库前会用 app.Key 加密
	app.Key = "0123456789abcdef0123456789abcdef"
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&biz.NotifyChannel{}, &biz.AlertRule{}, &biz.Alert{}, &biz.Setting{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// 渠道配置以原始 JSON 存取，读回后应与写入完全一致
func TestNotifyChannelConfigRoundTrip(t *testing.T) {
	repo := &notifyChannelRepo{db: newDBForTest(t)}

	config := json.RawMessage(`{"host":"smtp.example.com","port":465,"encryption":"ssl","to":["a@b.c"]}`)
	channel := &biz.NotifyChannel{Name: "mail", Type: "smtp", Config: config, Enabled: true}
	if err := repo.Create(channel); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(channel.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var want, actual map[string]any
	if err = json.Unmarshal(config, &want); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(got.Config, &actual); err != nil {
		t.Fatalf("unmarshal stored config %q: %v", string(got.Config), err)
	}
	if len(actual) != len(want) || actual["host"] != want["host"] {
		t.Fatalf("config mismatch: %s", string(got.Config))
	}
}

// 告警规则的渠道列表以 JSON 序列化存储
func TestAlertRuleChannelsRoundTrip(t *testing.T) {
	repo := &alertRepo{db: newDBForTest(t)}

	rule := &biz.AlertRule{Name: "cpu", Type: biz.AlertTypeCPU, Operator: biz.AlertOperatorGT, Threshold: 90, Duration: 3, Silence: 30, Channels: []uint{1, 2}, Enabled: true}
	if err := repo.CreateRule(rule); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetRule(rule.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Channels) != 2 || got.Channels[0] != 1 || got.Channels[1] != 2 {
		t.Fatalf("channels mismatch: %v", got.Channels)
	}
}

// 删除渠道后，告警规则与事件设置中的引用应一并清除，否则通知会静默失效
func TestNotifyChannelDeleteCleansReferences(t *testing.T) {
	db := newDBForTest(t)
	channels := &notifyChannelRepo{db: db}
	alerts := &alertRepo{db: db}

	kept := &biz.NotifyChannel{Name: "kept", Type: "smtp", Config: json.RawMessage(`{}`), Enabled: true}
	if err := channels.Create(kept); err != nil {
		t.Fatalf("create kept: %v", err)
	}
	removed := &biz.NotifyChannel{Name: "removed", Type: "smtp", Config: json.RawMessage(`{}`), Enabled: true}
	if err := channels.Create(removed); err != nil {
		t.Fatalf("create removed: %v", err)
	}

	rule := &biz.AlertRule{Name: "cpu", Type: biz.AlertTypeCPU, Operator: biz.AlertOperatorGT, Threshold: 90, Duration: 1, Channels: []uint{kept.ID, removed.ID}, Enabled: true}
	if err := alerts.CreateRule(rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	events, err := json.Marshal([]uint{kept.ID, removed.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&biz.Setting{Key: biz.SettingKeyNotifyEventChannels, Value: string(events)}).Error; err != nil {
		t.Fatal(err)
	}

	if err = channels.Delete(removed.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := alerts.GetRule(rule.ID)
	if err != nil {
		t.Fatalf("get rule: %v", err)
	}
	if len(got.Channels) != 1 || got.Channels[0] != kept.ID {
		t.Fatalf("rule channels not cleaned: %v", got.Channels)
	}

	setting := new(biz.Setting)
	if err = db.Where("key = ?", biz.SettingKeyNotifyEventChannels).First(setting).Error; err != nil {
		t.Fatalf("get setting: %v", err)
	}
	var remain []uint
	if err = json.Unmarshal([]byte(setting.Value), &remain); err != nil {
		t.Fatal(err)
	}
	if len(remain) != 1 || remain[0] != kept.ID {
		t.Fatalf("event channels not cleaned: %v", remain)
	}
}
