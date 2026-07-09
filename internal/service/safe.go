package service

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
)

type SafeService struct {
	safeRepo *biz.SafeUsecase
}

func NewSafeService(i do.Injector) (*SafeService, error) {
	return &SafeService{
		safeRepo: do.MustInvoke[*biz.SafeUsecase](i),
	}, nil
}

func (s *SafeService) GetPingStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.safeRepo.GetPingStatus()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, status)
}

func (s *SafeService) UpdatePingStatus(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.SafeUpdatePingStatus](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.safeRepo.UpdatePingStatus(r.Context(), req.Status); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
