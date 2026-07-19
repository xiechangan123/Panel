package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/tools"
)

type TamperService struct {
	t          *gotext.Locale
	tamperRepo *biz.TamperUsecase
}

func NewTamperService(i do.Injector) (*TamperService, error) {
	return &TamperService{
		t:          do.MustInvoke[*gotext.Locale](i),
		tamperRepo: do.MustInvoke[*biz.TamperUsecase](i),
	}, nil
}

// Status 防篡改运行状态与环境检测
func (s *TamperService) Status(w http.ResponseWriter, r *http.Request) {
	setting, err := s.tamperRepo.GetSetting()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, chix.M{
		"supported": s.tamperRepo.Supported(),
		"setting":   setting,
		"stats":     s.tamperRepo.Stats(),
		"ebpf":      s.tamperRepo.DetectEBPF(),
	})
}

// GetSetting 获取全局设置
func (s *TamperService) GetSetting(w http.ResponseWriter, r *http.Request) {
	setting, err := s.tamperRepo.GetSetting()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, setting)
}

// SaveSetting 保存全局设置(含开关,立即生效)
func (s *TamperService) SaveSetting(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TamperSetting](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	if !s.tamperRepo.Supported() {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("tamper protection is only supported on Linux"))
		return
	}
	// 启用 eBPF 模式前校验可用性
	if req.Enabled && req.Mode == "ebpf" {
		if st := s.tamperRepo.DetectEBPF(); !st.Available {
			Error(w, http.StatusUnprocessableEntity, s.t.Get("eBPF mode unavailable: %s", st.Reason))
			return
		}
	}

	if err = s.tamperRepo.SaveSetting(&biz.TamperSetting{
		Enabled:       req.Enabled,
		Mode:          req.Mode,
		BlockNewFiles: req.BlockNewFiles,
		LogDays:       req.LogDays,
	}); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
}

// ListRules 保护规则列表
func (s *TamperService) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.tamperRepo.ListRules()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	paged, total := Paginate(r, rules)
	Success(w, chix.M{"total": total, "items": paged})
}

// CreateRule 新增保护规则
func (s *TamperService) CreateRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TamperRuleCreate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	if err = s.tamperRepo.CreateRule(&biz.TamperRule{
		Name:     req.Name,
		Path:     req.Path,
		Exts:     req.Exts,
		Excludes: req.Excludes,
		Enabled:  req.Enabled,
	}); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
}

// UpdateRule 更新保护规则
func (s *TamperService) UpdateRule(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.TamperRuleUpdate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	rule, err := s.tamperRepo.GetRule(req.ID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	rule.Path = req.Path
	rule.Exts = req.Exts
	rule.Excludes = req.Excludes
	rule.Enabled = req.Enabled
	if err = s.tamperRepo.UpdateRule(rule); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
}

// DeleteRule 删除保护规则
func (s *TamperService) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := cast.ToUint(chi.URLParam(r, "id"))
	if err := s.tamperRepo.DeleteRule(id); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
}

// ListLogs 拦截日志
func (s *TamperService) ListLogs(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.Paginate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	logs, total, err := s.tamperRepo.ListLogs(req.Page, req.Limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, chix.M{"total": total, "items": logs})
}

// ClearLogs 清空拦截日志
func (s *TamperService) ClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := s.tamperRepo.ClearLogs(); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
}

// ActivateEBPF 修改 grub 激活 bpf LSM 并重启系统
func (s *TamperService) ActivateEBPF(w http.ResponseWriter, r *http.Request) {
	if err := s.tamperRepo.EnableBPFLSMGrub(); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	Success(w, nil)
	// 响应后重启系统使 bpf LSM 生效
	tools.RestartServer()
}
