package openlitespeed

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
	"github.com/acepanel/panel/v3/pkg/webserver"
	"github.com/acepanel/panel/v3/pkg/webserver/openlitespeed"
)

// rtReport 实时状态报告，由 OpenLiteSpeed 每 10 秒刷新
const rtReport = "/tmp/lshttpd/.rtreport"

type App struct {
	t *gotext.Locale
}

func NewApp(t *gotext.Locale) *App {
	return &App{
		t: t,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/load", s.Load)
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.SaveConfig)
	r.Get("/error_log", s.ErrorLog)
	r.Post("/clear_error_log", s.ClearErrorLog)
	r.Get("/php", s.PHPList)
	r.Post("/php", s.SetPHP)
	r.Get("/realip", s.GetRealIP)
	r.Post("/realip", s.SetRealIP)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("openlitespeed")
	return types.AggregateAppStatus(ok)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(openlitespeed.ServerRoot + "/conf/httpd_config.conf")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, config)
}

func (s *App) SaveConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(openlitespeed.ServerRoot+"/conf/httpd_config.conf", req.Config, 0600); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = s.reload(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) ErrorLog(w http.ResponseWriter, r *http.Request) {
	service.Success(w, openlitespeed.ServerRoot+"/logs/error.log")
}

func (s *App) ClearErrorLog(w http.ResponseWriter, r *http.Request) {
	if _, err := shell.Execf("cat /dev/null > %s/logs/error.log", openlitespeed.ServerRoot); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	report, err := io.Read(rtReport)
	if err != nil {
		service.Success(w, []types.NV{})
		return
	}

	var data []types.NV
	fields := []struct {
		key  string
		name string
	}{
		{"UPTIME", s.t.Get("Uptime")},
		{"PLAINCONN", s.t.Get("Connections")},
		{"SSLCONN", s.t.Get("SSL Connections")},
		{"IDLECONN", s.t.Get("Idle Connections")},
		{"REQ_PROCESSING", s.t.Get("Processing Requests")},
		{"REQ_PER_SEC", s.t.Get("Requests/sec")},
		{"TOT_REQS", s.t.Get("Total Requests")},
		{"BPS_IN", s.t.Get("Inbound KB/s")},
		{"BPS_OUT", s.t.Get("Outbound KB/s")},
	}
	for _, field := range fields {
		if match := regexp.MustCompile(field.key + `:\s*([^,\n]+)`).FindStringSubmatch(report); len(match) == 2 {
			data = append(data, types.NV{Name: field.name, Value: match[1]})
		}
	}

	service.Success(w, data)
}

// PHPList 列出已安装 PHP 版本的运行协议
func (s *App) PHPList(w http.ResponseWriter, r *http.Request) {
	matches, _ := filepath.Glob(filepath.Join(app.Root, "server", "php", "*", "bin", "php"))
	list := make([]PHPProtocol, 0, len(matches))
	for _, match := range matches {
		version, err := strconv.ParseUint(filepath.Base(filepath.Dir(filepath.Dir(match))), 10, 32)
		if err != nil {
			continue
		}
		_, lsphpErr := os.Stat(openlitespeed.LSPHPPath(uint(version)))
		list = append(list, PHPProtocol{
			Version: uint(version),
			LSPHP:   lsphpErr == nil,
			LSAPI:   openlitespeed.LSAPIEnabled(uint(version)),
		})
	}

	service.Success(w, list)
}

// SetPHP 切换 PHP 版本运行协议，lsphp 不存在时提示重装 PHP
func (s *App) SetPHP(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[SetPHP](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if req.LSAPI {
		if _, err = os.Stat(openlitespeed.LSPHPPath(req.Version)); err != nil {
			service.Error(w, http.StatusUnprocessableEntity, s.t.Get("lsphp not found, please reinstall PHP %d to enable LSAPI", req.Version))
			return
		}
	}
	if err = openlitespeed.SetLSAPI(req.Version, req.LSAPI); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if err = s.reload(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) GetRealIP(w http.ResponseWriter, r *http.Request) {
	realIP, err := openlitespeed.GetRealIP()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, realIP)
}

func (s *App) SetRealIP(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[SetRealIP](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = openlitespeed.SetRealIP(openlitespeed.RealIP{Enabled: req.Enabled, Trusted: req.Trusted}); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if err = s.reload(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// reload 经方言重载，重载前会 Sync
func (s *App) reload() error {
	d, err := webserver.Get(webserver.TypeOpenLiteSpeed)
	if err != nil {
		return err
	}
	if err = d.Reload(); err != nil {
		return fmt.Errorf("%s", s.t.Get("failed to reload openlitespeed: %v", err))
	}

	return nil
}
