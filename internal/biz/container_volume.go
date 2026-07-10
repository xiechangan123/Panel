package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerVolumeRepo interface {
	List(sock string) ([]types.ContainerVolume, error)
	Create(sock string, req *request.ContainerVolumeCreate) (string, error)
	Remove(sock string, id string) error
	Prune(sock string) error
}

type ContainerVolumeUsecase struct {
	repo    ContainerVolumeRepo
	setting SettingRepo
}

func NewContainerVolumeUsecase(repo ContainerVolumeRepo, setting SettingRepo) *ContainerVolumeUsecase {
	return &ContainerVolumeUsecase{repo: repo, setting: setting}
}

func (uc *ContainerVolumeUsecase) List() ([]types.ContainerVolume, error) {
	sock := containerSock(uc.setting)
	return uc.repo.List(sock)
}

func (uc *ContainerVolumeUsecase) Create(req *request.ContainerVolumeCreate) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Create(sock, req)
}

func (uc *ContainerVolumeUsecase) Remove(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(sock, id)
}

func (uc *ContainerVolumeUsecase) Prune() error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(sock)
}
