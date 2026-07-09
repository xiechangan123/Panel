package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerImageRepo interface {
	List() ([]types.ContainerImage, error)
	Exist(name string) (bool, error)
	Pull(req *request.ContainerImagePull) error
	Remove(id string) error
	Prune() error
}

type ContainerImageUsecase struct {
	repo ContainerImageRepo
}

func NewContainerImageUsecase(repo ContainerImageRepo) *ContainerImageUsecase {
	return &ContainerImageUsecase{repo: repo}
}

func (uc *ContainerImageUsecase) List() ([]types.ContainerImage, error) {
	return uc.repo.List()
}

func (uc *ContainerImageUsecase) Exist(name string) (bool, error) {
	return uc.repo.Exist(name)
}

func (uc *ContainerImageUsecase) Pull(req *request.ContainerImagePull) error {
	return uc.repo.Pull(req)
}

func (uc *ContainerImageUsecase) Remove(id string) error {
	return uc.repo.Remove(id)
}

func (uc *ContainerImageUsecase) Prune() error {
	return uc.repo.Prune()
}
