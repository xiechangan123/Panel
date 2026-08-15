package service

import (
	"net/http"

	"github.com/libtnb/chix/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
)

type DatabaseRedisService struct {
	repo *biz.DatabaseRedisUsecase
}

func NewDatabaseRedisService(databaseRedisUsecase *biz.DatabaseRedisUsecase) *DatabaseRedisService {
	return &DatabaseRedisService{repo: databaseRedisUsecase}
}

func (s *DatabaseRedisService) Databases(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisDatabases](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	count, err := s.repo.Databases(r.Context(), req)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, count)
}

func (s *DatabaseRedisService) Data(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisData](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	items, total, err := s.repo.Data(r.Context(), req)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, chix.M{
		"total": total,
		"items": items,
	})
}

func (s *DatabaseRedisService) KeyGet(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisKeyGet](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	kv, err := s.repo.KeyGet(r.Context(), req)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, kv)
}

func (s *DatabaseRedisService) KeySet(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisKeySet](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.repo.KeySet(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *DatabaseRedisService) KeyDelete(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisKeyDelete](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.repo.KeyDelete(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *DatabaseRedisService) KeyTTL(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisKeyTTL](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.repo.KeyTTL(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *DatabaseRedisService) KeyRename(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisKeyRename](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.repo.KeyRename(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *DatabaseRedisService) Clear(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.DatabaseRedisClear](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.repo.Clear(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
