package service

import (
	"net/http"

	"github.com/libtnb/chix/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
)

type ContainerImageService struct {
	containerImageRepo *biz.ContainerImageUsecase
}

func NewContainerImageService(containerImageUsecase *biz.ContainerImageUsecase) *ContainerImageService {
	return &ContainerImageService{
		containerImageRepo: containerImageUsecase,
	}
}

func (s *ContainerImageService) List(w http.ResponseWriter, r *http.Request) {
	images, err := s.containerImageRepo.List()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := Paginate(r, images)

	Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

func (s *ContainerImageService) Exist(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerImagePull](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	exist, err := s.containerImageRepo.Exist(req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, exist)
}

func (s *ContainerImageService) Pull(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerImagePull](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if req.Background {
		if err = s.containerImageRepo.PullBackground(req); err != nil {
			Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
		Success(w, nil)
		return
	}

	if err = s.containerImageRepo.Pull(req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *ContainerImageService) Remove(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ContainerImageID](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.containerImageRepo.Remove(req.ID); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *ContainerImageService) Prune(w http.ResponseWriter, r *http.Request) {
	if err := s.containerImageRepo.Prune(); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
