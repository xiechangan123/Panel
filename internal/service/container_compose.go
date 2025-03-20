package service

import (
	"net/http"

	"github.com/go-rat/chix"

	"github.com/tnb-labs/panel/internal/biz"
	"github.com/tnb-labs/panel/internal/http/request"
)

type ContainerComposeService struct {
	containerComposeRepo biz.ContainerComposeRepo
}

func NewContainerComposeService(containerCompose biz.ContainerComposeRepo) *ContainerComposeService {
	return &ContainerComposeService{
		containerComposeRepo: containerCompose,
	}
}

func (s *ContainerComposeService) List(w http.ResponseWriter, r *http.Request) {
	files, err := s.containerComposeRepo.List()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := Paginate(r, files)

	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *ContainerComposeService) Get(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerComposeGet](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	compose, env, err := s.containerComposeRepo.Get(req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, chix.M{
		"compose": compose,
		"env":     env,
	})
}

func (s *ContainerComposeService) Create(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerComposeCreate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.containerComposeRepo.Create(req.Name, req.Compose, req.Env); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *ContainerComposeService) Up(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerComposeUp](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.containerComposeRepo.Up(req.Name, req.Force); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *ContainerComposeService) Down(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerComposeDown](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.containerComposeRepo.Down(req.Name); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *ContainerComposeService) Remove(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerComposeRemove](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.containerComposeRepo.Remove(req.Name); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
