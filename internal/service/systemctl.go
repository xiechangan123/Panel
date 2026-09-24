package service

import (
	"net/http"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

type SystemctlService struct {
	t *gotext.Locale
}

func NewSystemctlService(t *gotext.Locale) *SystemctlService {
	return &SystemctlService{
		t: t,
	}
}

func (s *SystemctlService) Units(w http.ResponseWriter, r *http.Request) {
	units, err := systemctl.ListUnits(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to list system units: %v", err))
		return
	}

	Success(w, units)
}

func (s *SystemctlService) Status(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	status, err := systemctl.Status(r.Context(), req.Service)
	if err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to get %s service running status: %v", req.Service, err))
		return
	}

	Success(w, status)
}

func (s *SystemctlService) IsEnabled(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	enabled, err := systemctl.IsEnabled(r.Context(), req.Service)
	if err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to get %s service enable status: %v", req.Service, err))
		return
	}

	Success(w, enabled)
}

func (s *SystemctlService) Enable(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Enable(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to enable %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

func (s *SystemctlService) Disable(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Disable(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to disable %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

func (s *SystemctlService) Restart(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Restart(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to restart %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

func (s *SystemctlService) Reload(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Reload(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to reload %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

func (s *SystemctlService) Start(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Start(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to start %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

func (s *SystemctlService) Stop(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.Stop(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to stop %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}

// ClearLog 清空指定 systemd 服务的 journald 日志
func (s *SystemctlService) ClearLog(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SystemctlService](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = systemctl.LogClear(r.Context(), req.Service); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to clear log for %s service: %v", req.Service, err))
		return
	}

	Success(w, nil)
}
