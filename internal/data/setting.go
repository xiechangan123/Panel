package data

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/go-rat/utils/hash"
	"github.com/knadh/koanf/v2"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/tnb-labs/panel/internal/app"
	"github.com/tnb-labs/panel/internal/biz"
	"github.com/tnb-labs/panel/internal/http/request"
	"github.com/tnb-labs/panel/pkg/cert"
	"github.com/tnb-labs/panel/pkg/firewall"
	"github.com/tnb-labs/panel/pkg/io"
	"github.com/tnb-labs/panel/pkg/os"
	"github.com/tnb-labs/panel/pkg/types"
)

type settingRepo struct {
	t    *gotext.Locale
	db   *gorm.DB
	conf *koanf.Koanf
	task biz.TaskRepo
}

func NewSettingRepo(t *gotext.Locale, db *gorm.DB, conf *koanf.Koanf, task biz.TaskRepo) biz.SettingRepo {
	return &settingRepo{
		t:    t,
		db:   db,
		conf: conf,
		task: task,
	}
}

func (r *settingRepo) Get(key biz.SettingKey, defaultValue ...string) (string, error) {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	if setting.Value == "" && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}

	return setting.Value, nil
}

func (r *settingRepo) GetBool(key biz.SettingKey, defaultValue ...bool) (bool, error) {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
	}

	if setting.Value == "" && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}

	return cast.ToBool(setting.Value), nil
}

func (r *settingRepo) GetInt(key biz.SettingKey, defaultValue ...int) (int, error) {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}

	if setting.Value == "" && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}

	return cast.ToInt(setting.Value), nil
}

func (r *settingRepo) GetSlice(key biz.SettingKey, defaultValue ...[]string) ([]string, error) {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 设置值为空时提前返回
	slice := make([]string, 0)
	if setting.Value == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return slice, nil
	}

	if err := json.Unmarshal([]byte(setting.Value), &slice); err != nil {
		return nil, err
	}
	if len(slice) == 0 && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}

	return slice, nil
}

func (r *settingRepo) Set(key biz.SettingKey, value string) error {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	setting.Key = key
	setting.Value = value
	return r.db.Save(setting).Error
}

func (r *settingRepo) SetSlice(key biz.SettingKey, value []string) error {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).First(setting).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	setting.Key = key
	if len(value) == 0 {
		setting.Value = "[]"
	} else {
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		setting.Value = string(b)
	}

	return r.db.Save(setting).Error
}

func (r *settingRepo) Delete(key biz.SettingKey) error {
	setting := new(biz.Setting)
	if err := r.db.Where("key = ?", key).Delete(setting).Error; err != nil {
		return err
	}

	return nil
}

func (r *settingRepo) GetPanelSetting(ctx context.Context) (*request.PanelSetting, error) {
	name, err := r.Get(biz.SettingKeyName)
	if err != nil {
		return nil, err
	}
	offlineMode, err := r.GetBool(biz.SettingKeyOfflineMode)
	if err != nil {
		return nil, err
	}
	autoUpdate, err := r.GetBool(biz.SettingKeyAutoUpdate)
	if err != nil {
		return nil, err
	}
	twoFA, err := r.GetBool(biz.SettingKeyTwoFA)
	if err != nil {
		return nil, err
	}
	loginTimeout, err := r.GetInt(biz.SettingKeyLoginTimeout)
	if err != nil {
		return nil, err
	}
	bindDomain, err := r.GetSlice(biz.SettingKeyBindDomain)
	if err != nil {
		return nil, err
	}
	bindIP, err := r.GetSlice(biz.SettingKeyBindIP)
	if err != nil {
		return nil, err
	}
	bindUA, err := r.GetSlice(biz.SettingKeyBindUA)
	if err != nil {
		return nil, err
	}

	websitePath, err := r.Get(biz.SettingKeyWebsitePath)
	if err != nil {
		return nil, err
	}
	backupPath, err := r.Get(biz.SettingKeyBackupPath)
	if err != nil {
		return nil, err
	}

	userID := cast.ToUint(ctx.Value("user_id"))
	user := new(biz.User)
	if err := r.db.Where("id = ?", userID).First(user).Error; err != nil {
		return nil, err
	}

	crt, err := io.Read(filepath.Join(app.Root, "panel/storage/cert.pem"))
	if err != nil {
		return nil, err
	}
	key, err := io.Read(filepath.Join(app.Root, "panel/storage/cert.key"))
	if err != nil {
		return nil, err
	}

	return &request.PanelSetting{
		Name:         name,
		Locale:       r.conf.String("app.locale"),
		Entrance:     r.conf.String("http.entrance"),
		OfflineMode:  offlineMode,
		AutoUpdate:   autoUpdate,
		TwoFA:        twoFA,
		LoginTimeout: loginTimeout,
		BindDomain:   bindDomain,
		BindIP:       bindIP,
		BindUA:       bindUA,
		WebsitePath:  websitePath,
		BackupPath:   backupPath,
		Username:     user.Username,
		Email:        user.Email,
		Port:         uint(r.conf.Int("http.port")),
		HTTPS:        r.conf.Bool("http.tls"),
		Cert:         crt,
		Key:          key,
	}, nil
}

func (r *settingRepo) UpdatePanelSetting(ctx context.Context, setting *request.PanelSetting) (bool, error) {
	if err := r.Set(biz.SettingKeyName, setting.Name); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyOfflineMode, cast.ToString(setting.OfflineMode)); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyAutoUpdate, cast.ToString(setting.AutoUpdate)); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyLoginTimeout, cast.ToString(setting.LoginTimeout)); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyTwoFA, cast.ToString(setting.TwoFA)); err != nil {
		return false, err
	}
	if err := r.SetSlice(biz.SettingKeyBindDomain, setting.BindDomain); err != nil {
		return false, err
	}
	if err := r.SetSlice(biz.SettingKeyBindIP, setting.BindIP); err != nil {
		return false, err
	}
	if err := r.SetSlice(biz.SettingKeyBindUA, setting.BindUA); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyWebsitePath, setting.WebsitePath); err != nil {
		return false, err
	}
	if err := r.Set(biz.SettingKeyBackupPath, setting.BackupPath); err != nil {
		return false, err
	}

	// 用户
	user := new(biz.User)
	userID := cast.ToUint(ctx.Value("user_id"))
	if err := r.db.Where("id = ?", userID).First(user).Error; err != nil {
		return false, err
	}

	user.Username = setting.Username
	user.Email = setting.Email
	if setting.Password != "" {
		value, err := hash.NewArgon2id().Make(setting.Password)
		if err != nil {
			return false, err
		}
		user.Password = value
	}
	if err := r.db.Save(user).Error; err != nil {
		return false, err
	}

	// 下面是需要需要重启的设置
	// 面板HTTPS
	restartFlag := false
	oldCert, _ := io.Read(filepath.Join(app.Root, "panel/storage/cert.pem"))
	oldKey, _ := io.Read(filepath.Join(app.Root, "panel/storage/cert.key"))
	if oldCert != setting.Cert || oldKey != setting.Key {
		if r.task.HasRunningTask() {
			return false, errors.New(r.t.Get("background task is running, modifying some settings is prohibited, please try again later"))
		}
		restartFlag = true
	}
	if _, err := cert.ParseCert(setting.Cert); err != nil {
		return false, errors.New(r.t.Get("failed to parse certificate: %v", err))
	}
	if _, err := cert.ParseKey(setting.Key); err != nil {
		return false, errors.New(r.t.Get("failed to parse private key: %v", err))
	}
	if err := io.Write(filepath.Join(app.Root, "panel/storage/cert.pem"), setting.Cert, 0644); err != nil {
		return false, err
	}
	if err := io.Write(filepath.Join(app.Root, "panel/storage/cert.key"), setting.Key, 0644); err != nil {
		return false, err
	}

	// 面板主配置
	config := new(types.PanelConfig)
	raw, err := io.Read("/usr/local/etc/panel/config.yml")
	if err != nil {
		return false, err
	}
	if err = yaml.Unmarshal([]byte(raw), config); err != nil {
		return false, err
	}

	if setting.Port != config.HTTP.Port {
		if os.TCPPortInUse(setting.Port) {
			return false, errors.New(r.t.Get("port is already in use"))
		}
	}

	config.App.Locale = setting.Locale
	config.HTTP.Port = setting.Port
	config.HTTP.Entrance = setting.Entrance
	config.HTTP.TLS = setting.HTTPS

	// 放行端口
	fw := firewall.NewFirewall()
	err = fw.Port(firewall.FireInfo{
		Type:      firewall.TypeNormal,
		PortStart: config.HTTP.Port,
		PortEnd:   config.HTTP.Port,
		Direction: firewall.DirectionIn,
		Strategy:  firewall.StrategyAccept,
	}, firewall.OperationAdd)
	if err != nil {
		return false, err
	}

	encoded, err := yaml.Marshal(config)
	if err != nil {
		return false, err
	}
	if raw != string(encoded) {
		if r.task.HasRunningTask() {
			return false, errors.New(r.t.Get("background task is running, modifying some settings is prohibited, please try again later"))
		}
		restartFlag = true
	}
	if err = io.Write("/usr/local/etc/panel/config.yml", string(encoded), 0644); err != nil {
		return false, err
	}

	return restartFlag, nil
}
