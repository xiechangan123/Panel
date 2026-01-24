package postgresql

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/acepanel/panel/pkg/db"
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/service"
	"github.com/acepanel/panel/pkg/io"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/systemctl"
	"github.com/acepanel/panel/pkg/types"
)

type App struct {
	t           *gotext.Locale
	settingRepo biz.SettingRepo
}

func NewApp(t *gotext.Locale, setting biz.SettingRepo) *App {
	return &App{
		t:           t,
		settingRepo: setting,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/user_config", s.GetUserConfig)
	r.Post("/user_config", s.UpdateUserConfig)
	r.Get("/load", s.Load)
	r.Get("/log", s.Log)
	r.Post("/clear_log", s.ClearLog)
	r.Get("/postgres_password", s.GetPostgresPassword)
	r.Post("/postgres_password", s.SetPostgresPassword)
}

// GetConfig 获取配置
func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	// 获取配置
	config, err := io.Read(fmt.Sprintf("%s/server/postgresql/data/postgresql.conf", app.Root))
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

	if err = io.Write(fmt.Sprintf("%s/server/postgresql/data/postgresql.conf", app.Root), req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Reload("postgresql"); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to reload PostgreSQL: %v", err))
		return
	}

	service.Success(w, nil)
}

// GetUserConfig 获取用户配置
func (s *App) GetUserConfig(w http.ResponseWriter, r *http.Request) {
	// 获取配置
	config, err := io.Read(fmt.Sprintf("%s/server/postgresql/data/pg_hba.conf", app.Root))
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

	if err = io.Write(fmt.Sprintf("%s/server/postgresql/data/pg_hba.conf", app.Root), req.Config, 0644); err != nil {
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

	if err = os.Setenv("PGPASSWORD", postgresPassword); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to set PGPASSWORD env: %v", err))
		return
	}

	start, err := shell.Execf(`psql -h 127.0.0.1 -U postgres -t -c "select pg_postmaster_start_time();" | head -1 | cut -d'.' -f1`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL start time: %v", err))
		return
	}
	pid, err := shell.Execf(`psql -h 127.0.0.1 -U postgres -t -c "select pg_backend_pid();"`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL backend pid: %v", err))
		return
	}
	process, err := shell.Execf(`ps aux | grep postgres | grep -v grep | wc -l`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL process: %v", err))
		return
	}
	connections, err := shell.Execf(`psql -h 127.0.0.1 -U postgres -t -c "SELECT count(*) FROM pg_stat_activity WHERE NOT pid=pg_backend_pid();"`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL connections: %v", err))
		return
	}
	storage, err := shell.Execf(`psql -h 127.0.0.1 -U postgres -t -c "select pg_size_pretty(pg_database_size('postgres'));"`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get PostgreSQL database size: %v", err))
		return
	}

	if err = os.Unsetenv("PGPASSWORD"); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to unset PGPASSWORD env: %v", err))
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

// Log 获取日志
func (s *App) Log(w http.ResponseWriter, r *http.Request) {
	service.Success(w, fmt.Sprintf("%s/server/postgresql/logs/postgresql-%s.log", app.Root, time.Now().Format(time.DateOnly)))
}

// ClearLog 清空日志
func (s *App) ClearLog(w http.ResponseWriter, r *http.Request) {
	if _, err := shell.Execf("rm -rf %s/server/postgresql/logs/postgresql-*.log", app.Root); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
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
	postgres, err := db.NewPostgres("postgres", oldPassword, "127.0.0.1", 5432)
	if err != nil {
		// 直接修改密码
		if _, err = shell.Execf(`su - postgres -c "psql -c \"ALTER USER postgres WITH PASSWORD '%s';\""`, req.Password); err != nil {
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

	service.Success(w, nil)
}
