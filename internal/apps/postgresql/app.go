package postgresql

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/db"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t                  *gotext.Locale
	settingRepo        biz.SettingRepo
	databaseServerRepo biz.DatabaseServerRepo
}

func NewApp(t *gotext.Locale, databaseServerRepo biz.DatabaseServerRepo, settingRepo biz.SettingRepo) *App {

	setting := settingRepo
	databaseServer := databaseServerRepo
	return &App{
		t:                  t,
		settingRepo:        setting,
		databaseServerRepo: databaseServer,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/user_config", s.GetUserConfig)
	r.Post("/user_config", s.UpdateUserConfig)
	r.Get("/load", s.Load)
	r.Get("/log", s.Log)
	r.Get("/postgres_password", s.GetPostgresPassword)
	r.Post("/postgres_password", s.SetPostgresPassword)
	r.Get("/config_tune", s.GetConfigTune)
	r.Post("/config_tune", s.UpdateConfigTune)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("postgresql")
	return types.AggregateAppStatus(ok)
}

// GetConfig 获取配置
func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	// 获取配置
	config, err := io.Read(s.configPath())
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, config)
}

// UpdateConfig 保存配置
func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	oldPort := db.PostgresPort(app.Root)
	if err = io.Write(s.configPath(), req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = s.applyConfig(req.Config, oldPort); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to apply PostgreSQL config: %v", err))
		return
	}

	service.Success(w, nil)
}

// GetUserConfig 获取用户配置
func (s *App) GetUserConfig(w http.ResponseWriter, r *http.Request) {
	// 获取配置
	config, err := io.Read(app.Root + "/server/postgresql/data/pg_hba.conf")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, config)
}

// UpdateUserConfig 保存用户配置
func (s *App) UpdateUserConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(app.Root+"/server/postgresql/data/pg_hba.conf", req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Reload("postgresql"); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to reload PostgreSQL: %v", err))
		return
	}

	service.Success(w, nil)
}

// Load 获取负载
func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	status, _ := systemctl.Status("postgresql")
	if !status {
		service.Success(w, []types.NV{})
		return
	}

	postgresPassword, err := s.settingRepo.Get(biz.SettingKeyPostgresPassword)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to load PostgreSQL postgres password: %v", err))
		return
	}

	env := []string{"PGPASSWORD=" + postgresPassword}
	port := db.PostgresPort(app.Root)
	start, err := shell.ExecfWithEnv(env, `psql -h 127.0.0.1 -p %d -U postgres -t -c "select pg_postmaster_start_time();" | head -1 | cut -d'.' -f1`, port)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL start time: %v", err))
		return
	}
	pid, err := shell.ExecfWithEnv(env, `psql -h 127.0.0.1 -p %d -U postgres -t -c "select pg_backend_pid();"`, port)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL backend pid: %v", err))
		return
	}
	process, err := shell.Execf(`ps aux | grep postgres | grep -v grep | wc -l`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL process: %v", err))
		return
	}
	connections, err := shell.ExecfWithEnv(env, `psql -h 127.0.0.1 -p %d -U postgres -t -c "SELECT count(*) FROM pg_stat_activity WHERE NOT pid=pg_backend_pid();"`, port)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL connections: %v", err))
		return
	}
	storage, err := shell.ExecfWithEnv(env, `psql -h 127.0.0.1 -p %d -U postgres -t -c "select pg_size_pretty(pg_database_size('postgres'));"`, port)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL database size: %v", err))
		return
	}

	data := []types.NV{
		{Name: s.t.Get("Start Time"), Value: start},
		{Name: s.t.Get("Process PID"), Value: pid},
		{Name: s.t.Get("Process Count"), Value: process},
		{Name: s.t.Get("Total Connections"), Value: connections},
		{Name: s.t.Get("Storage Usage"), Value: storage},
	}

	service.Success(w, data)
}

// Log 获取应用日志路径列表
func (s *App) Log(w http.ResponseWriter, r *http.Request) {
	paths, err := filepath.Glob(app.Root + "/server/postgresql/logs/postgresql-*.log")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, paths)
}

// GetPostgresPassword 获取 postgres 用户密码
func (s *App) GetPostgresPassword(w http.ResponseWriter, r *http.Request) {
	password, err := s.settingRepo.Get(biz.SettingKeyPostgresPassword)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get postgres password: %v", err))
		return
	}

	service.Success(w, password)
}

// SetPostgresPassword 设置 postgres 用户密码
func (s *App) SetPostgresPassword(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[SetPostgresPassword](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	oldPassword, _ := s.settingRepo.Get(biz.SettingKeyPostgresPassword)
	port := db.PostgresPort(app.Root)
	postgres, err := db.NewPostgres(r.Context(), "postgres", oldPassword, "127.0.0.1", port)
	if err != nil {
		// 直接修改密码
		if _, err = shell.Execf(`su - postgres -c "psql -p %d -c \"ALTER USER postgres WITH PASSWORD '%s';\""`, port, req.Password); err != nil {
			service.Error(w, http.StatusInternalServerError, s.t.Get("failed to set postgres password: %v", err))
			return
		}
	} else {
		defer postgres.Close()
		if err = postgres.UserPassword("postgres", req.Password); err != nil {
			service.Error(w, http.StatusInternalServerError, s.t.Get("failed to set postgres password: %v", err))
			return
		}
	}

	if err = s.settingRepo.Set(biz.SettingKeyPostgresPassword, req.Password); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to set postgres password: %v", err))
		return
	}

	_ = s.databaseServerRepo.UpdatePassword("local_postgresql", req.Password)

	service.Success(w, nil)
}

// GetConfigTune 获取 PostgreSQL 配置调整参数
func (s *App) GetConfigTune(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(s.configPath())
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	tune := ConfigTune{
		// 连接设置
		ListenAddresses:              confval.Postgres.Get(config, "listen_addresses"),
		Port:                         confval.Postgres.Get(config, "port"),
		MaxConnections:               confval.Postgres.Get(config, "max_connections"),
		SuperuserReservedConnections: confval.Postgres.Get(config, "superuser_reserved_connections"),
		// 内存设置
		SharedBuffers:      confval.Postgres.Get(config, "shared_buffers"),
		WorkMem:            confval.Postgres.Get(config, "work_mem"),
		MaintenanceWorkMem: confval.Postgres.Get(config, "maintenance_work_mem"),
		EffectiveCacheSize: confval.Postgres.Get(config, "effective_cache_size"),
		HugePages:          confval.Postgres.Get(config, "huge_pages"),
		// WAL 设置
		WalLevel:                   confval.Postgres.Get(config, "wal_level"),
		WalBuffers:                 confval.Postgres.Get(config, "wal_buffers"),
		MaxWalSize:                 confval.Postgres.Get(config, "max_wal_size"),
		MinWalSize:                 confval.Postgres.Get(config, "min_wal_size"),
		CheckpointCompletionTarget: confval.Postgres.Get(config, "checkpoint_completion_target"),
		// 查询优化
		DefaultStatisticsTarget: confval.Postgres.Get(config, "default_statistics_target"),
		RandomPageCost:          confval.Postgres.Get(config, "random_page_cost"),
		EffectiveIoConcurrency:  confval.Postgres.Get(config, "effective_io_concurrency"),
		// 日志设置
		LogDestination:          confval.Postgres.Get(config, "log_destination"),
		LogMinDurationStatement: confval.Postgres.Get(config, "log_min_duration_statement"),
		LogTimezone:             confval.Postgres.Get(config, "log_timezone"),
		// IO 设置
		IoMethod: confval.Postgres.Get(config, "io_method"),
	}

	service.Success(w, tune)
}

// UpdateConfigTune 更新 PostgreSQL 配置调整参数
func (s *App) UpdateConfigTune(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ConfigTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	confPath := s.configPath()
	oldPort := db.PostgresPort(app.Root)
	config, err := io.Read(confPath)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 更新连接设置
	config = confval.Postgres.Set(config, "listen_addresses", req.ListenAddresses)
	config = confval.Postgres.Set(config, "port", req.Port)
	config = confval.Postgres.Set(config, "max_connections", req.MaxConnections)
	config = confval.Postgres.Set(config, "superuser_reserved_connections", req.SuperuserReservedConnections)
	// 更新内存设置
	config = confval.Postgres.Set(config, "shared_buffers", req.SharedBuffers)
	config = confval.Postgres.Set(config, "work_mem", req.WorkMem)
	config = confval.Postgres.Set(config, "maintenance_work_mem", req.MaintenanceWorkMem)
	config = confval.Postgres.Set(config, "effective_cache_size", req.EffectiveCacheSize)
	config = confval.Postgres.Set(config, "huge_pages", req.HugePages)
	// 更新 WAL 设置
	config = confval.Postgres.Set(config, "wal_level", req.WalLevel)
	config = confval.Postgres.Set(config, "wal_buffers", req.WalBuffers)
	config = confval.Postgres.Set(config, "max_wal_size", req.MaxWalSize)
	config = confval.Postgres.Set(config, "min_wal_size", req.MinWalSize)
	config = confval.Postgres.Set(config, "checkpoint_completion_target", req.CheckpointCompletionTarget)
	// 更新查询优化
	config = confval.Postgres.Set(config, "default_statistics_target", req.DefaultStatisticsTarget)
	config = confval.Postgres.Set(config, "random_page_cost", req.RandomPageCost)
	config = confval.Postgres.Set(config, "effective_io_concurrency", req.EffectiveIoConcurrency)
	// 更新日志设置
	config = confval.Postgres.Set(config, "log_destination", req.LogDestination)
	config = confval.Postgres.Set(config, "log_min_duration_statement", req.LogMinDurationStatement)
	config = confval.Postgres.Set(config, "log_timezone", req.LogTimezone)
	// 更新 IO 设置
	config = confval.Postgres.Set(config, "io_method", req.IoMethod)

	if err = io.Write(confPath, config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = s.applyConfig(config, oldPort); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to apply PostgreSQL config: %v", err))
		return
	}

	service.Success(w, nil)
}

func (s *App) configPath() string {
	return app.Root + "/server/postgresql/data/postgresql.conf"
}

// parsePort 从 config 内容解析端口，未配置时返回默认值
func (s *App) parsePort(config string) uint {
	port := cast.ToUint(confval.Postgres.Get(config, "port"))
	if port == 0 {
		return 5432
	}
	return port
}

// applyConfig 让 PostgreSQL 配置生效
func (s *App) applyConfig(newConfig string, oldPort uint) error {
	newPort := s.parsePort(newConfig)
	if oldPort == newPort {
		return systemctl.Reload("postgresql")
	}
	if err := systemctl.Restart("postgresql"); err != nil {
		return err
	}
	return s.databaseServerRepo.UpdatePort("local_postgresql", newPort)
}
