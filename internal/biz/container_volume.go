package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerVolumeRepo interface {
	List() ([]types.ContainerVolume, error)
	Create(req *request.ContainerVolumeCreate) (string, error)
	Remove(id string) error
	Prune() error
}

type ContainerVolumeUsecase struct {
	repo ContainerVolumeRepo
}

func NewContainerVolumeUsecase(repo ContainerVolumeRepo) *ContainerVolumeUsecase {
	return &ContainerVolumeUsecase{repo: repo}
}

func (uc *ContainerVolumeUsecase) List() ([]types.ContainerVolume, error) {
	return uc.repo.List()
}

func (uc *ContainerVolumeUsecase) Create(req *request.ContainerVolumeCreate) (string, error) {
	return uc.repo.Create(req)
}

func (uc *ContainerVolumeUsecase) Remove(id string) error {
	return uc.repo.Remove(id)
}

func (uc *ContainerVolumeUsecase) Prune() error {
	return uc.repo.Prune()
}
