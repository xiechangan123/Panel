package redis

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps/common"
	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t                  *gotext.Locale
	databaseServerRepo biz.DatabaseServerRepo
	slug               string // 服务名与配置目录名，如 redis、valkey
	name               string // 展示名，如 Redis、Valkey
}

func NewApp(t *gotext.Locale, databaseServerRepo biz.DatabaseServerRepo) *App {
	return New("redis", "Redis", t, databaseServerRepo)
}

func New(slug, name string, t *gotext.Locale, databaseServerRepo biz.DatabaseServerRepo) *App {
	return &App{
		t:                  t,
		databaseServerRepo: databaseServerRepo,
		slug:               slug,
		name:               name,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/load", s.Load)
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/config_tune", s.GetConfigTune)
	r.Post("/config_tune", s.UpdateConfigTune)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status(s.slug)
	return types.AggregateAppStatus(ok)
}

func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	status, err := systemctl.Status(s.slug)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get %s status: %v", s.name, err))
		return
	}
	if !status {
		service.Success(w, []types.NV{})
		return
	}

	// 检查密码
	withPassword := ""
	config, err := io.Read(s.confPath())
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	re := regexp.MustCompile(`(?m)^requirepass\s+(.+)`)
	matches := re.FindStringSubmatch(config)
	if len(matches) == 2 {
		withPassword = " -a " + matches[1]
	}

	raw, err := shell.Execf("%s%s info", s.slug+"-cli", withPassword)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get %s info: %v", s.name, err))
		return
	}

	dataRaw := lo.SliceToMap(strings.Split(raw, "\n"), func(item string) (string, string) {
		parts := strings.Split(item, ":")
		if len(parts) != 2 {
			return "", ""
		}
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	})

	data := []types.NV{
		{Name: s.t.Get("TCP Port"), Value: dataRaw["tcp_port"]},
		{Name: s.t.Get("Uptime in Days"), Value: dataRaw["uptime_in_days"]},
		{Name: s.t.Get("Connected Clients"), Value: dataRaw["connected_clients"]},
		{Name: s.t.Get("Total Allocated Memory"), Value: dataRaw["used_memory_human"]},
		{Name: s.t.Get("Total Memory Usage"), Value: dataRaw["used_memory_rss_human"]},
		{Name: s.t.Get("Peak Memory Usage"), Value: dataRaw["used_memory_peak_human"]},
		{Name: s.t.Get("Memory Fragmentation Ratio"), Value: dataRaw["mem_fragmentation_ratio"]},
		{Name: s.t.Get("Total Connections Received"), Value: dataRaw["total_connections_received"]},
		{Name: s.t.Get("Total Commands Processed"), Value: dataRaw["total_commands_processed"]},
		{Name: s.t.Get("Commands Per Second"), Value: dataRaw["instantaneous_ops_per_sec"]},
		{Name: s.t.Get("Keyspace Hits"), Value: dataRaw["keyspace_hits"]},
		{Name: s.t.Get("Keyspace Misses"), Value: dataRaw["keyspace_misses"]},
		{Name: s.t.Get("Latest Fork Time (ms)"), Value: dataRaw["latest_fork_usec"]},
	}

	service.Success(w, data)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	common.ServeConfig(w, s.confPath())
}

func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	common.SaveConfig(w, r, s.confPath(), s.slug)
}

// GetConfigTune 获取配置调整参数
func (s *App) GetConfigTune(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(s.confPath())
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	tune := ConfigTune{
		Bind:            confval.Directive.Get(config, "bind"),
		Port:            confval.Directive.Get(config, "port"),
		Databases:       confval.Directive.Get(config, "databases"),
		Requirepass:     confval.Directive.Get(config, "requirepass"),
		Timeout:         confval.Directive.Get(config, "timeout"),
		TCPKeepalive:    confval.Directive.Get(config, "tcp-keepalive"),
		Maxmemory:       confval.Directive.Get(config, "maxmemory"),
		MaxmemoryPolicy: confval.Directive.Get(config, "maxmemory-policy"),
		Appendonly:      confval.Directive.Get(config, "appendonly"),
		Appendfsync:     confval.Directive.Get(config, "appendfsync"),
	}

	service.Success(w, tune)
}

// UpdateConfigTune 更新配置调整参数
func (s *App) UpdateConfigTune(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ConfigTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	confPath := s.confPath()
	config, err := io.Read(confPath)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	config = confval.Directive.Set(config, "bind", req.Bind)
	config = confval.Directive.Set(config, "port", req.Port)
	config = confval.Directive.Set(config, "databases", req.Databases)
	config = confval.Directive.Set(config, "requirepass", req.Requirepass)
	config = confval.Directive.Set(config, "timeout", req.Timeout)
	config = confval.Directive.Set(config, "tcp-keepalive", req.TCPKeepalive)
	config = confval.Directive.Set(config, "maxmemory", req.Maxmemory)
	config = confval.Directive.Set(config, "maxmemory-policy", req.MaxmemoryPolicy)
	config = confval.Directive.Set(config, "appendonly", req.Appendonly)
	config = confval.Directive.Set(config, "appendfsync", req.Appendfsync)

	if err = io.Write(confPath, config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart(s.slug); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 同步密码到数据库服务器记录
	_ = s.databaseServerRepo.UpdatePassword("local_"+s.slug, req.Requirepass)

	service.Success(w, nil)
}

func (s *App) confPath() string {
	return filepath.Join(app.Root, "server", s.slug, s.slug+".conf")
}
