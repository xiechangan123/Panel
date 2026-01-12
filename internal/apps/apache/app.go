package apache

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/service"
	"github.com/acepanel/panel/pkg/io"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/systemctl"
	"github.com/acepanel/panel/pkg/tools"
	"github.com/acepanel/panel/pkg/types"
)

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
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(fmt.Sprintf("%s/server/apache/conf/httpd.conf", app.Root))
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

	if err = io.Write(fmt.Sprintf("%s/server/apache/conf/httpd.conf", app.Root), req.Config, 0600); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Reload("apache"); err != nil {
		_, err = shell.Execf("%s/server/apache/bin/apachectl configtest", app.Root)
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to reload apache: %v", err))
		return
	}

	service.Success(w, nil)
}

func (s *App) ErrorLog(w http.ResponseWriter, r *http.Request) {
	service.Success(w, fmt.Sprintf("%s/%s", app.Root, "server/apache/logs/error_log"))
}

func (s *App) ClearErrorLog(w http.ResponseWriter, r *http.Request) {
	if _, err := shell.Execf("cat /dev/null > %s/%s", app.Root, "server/apache/logs/error_log"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	status, err := shell.Execf("curl -s http://127.0.0.1/server_status?auto 2>/dev/null || true")
	if err != nil {
		service.Success(w, []types.NV{})
		return
	}

	var data []types.NV

	workers, err := shell.Execf("ps aux | grep httpd | grep -v grep | wc -l")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get apache workers: %v", err))
		return
	}
	data = append(data, types.NV{
		Name:  s.t.Get("Workers"),
		Value: workers,
	})

	out, err := shell.Execf("ps aux | grep httpd | grep -v grep | awk '{memsum+=$6};END {print memsum}'")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get apache workers: %v", err))
		return
	}
	mem := tools.FormatBytes(cast.ToFloat64(out))
	data = append(data, types.NV{
		Name:  s.t.Get("Memory"),
		Value: mem,
	})

	// Parse server-status output
	if match := regexp.MustCompile(`Total Accesses:\s*(\d+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Total Accesses"),
			Value: match[1],
		})
	}

	if match := regexp.MustCompile(`Total kBytes:\s*(\d+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Total Traffic"),
			Value: tools.FormatBytes(cast.ToFloat64(match[1]) * 1024),
		})
	}

	if match := regexp.MustCompile(`BusyWorkers:\s*(\d+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Busy Workers"),
			Value: match[1],
		})
	}

	if match := regexp.MustCompile(`IdleWorkers:\s*(\d+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Idle Workers"),
			Value: match[1],
		})
	}

	if match := regexp.MustCompile(`ReqPerSec:\s*([\d.]+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Requests/sec"),
			Value: match[1],
		})
	}

	if match := regexp.MustCompile(`BytesPerSec:\s*([\d.]+)`).FindStringSubmatch(status); len(match) == 2 {
		data = append(data, types.NV{
			Name:  s.t.Get("Bytes/sec"),
			Value: tools.FormatBytes(cast.ToFloat64(match[1])),
		})
	}

	service.Success(w, data)
}
