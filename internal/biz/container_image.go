package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerImageRepo interface {
	List(sock string) ([]types.ContainerImage, error)
	Exist(sock string, name string) (bool, error)
	Pull(sock string, req *request.ContainerImagePull) error
	Remove(sock string, id string) error
	Prune(sock string) error
}

type ContainerImageUsecase struct {
	repo    ContainerImageRepo
	setting SettingRepo
}

func NewContainerImageUsecase(repo ContainerImageRepo, setting SettingRepo) *ContainerImageUsecase {
	return &ContainerImageUsecase{repo: repo, setting: setting}
}

func (uc *ContainerImageUsecase) List() ([]types.ContainerImage, error) {
	sock := containerSock(uc.setting)
	return uc.repo.List(sock)
}

func (uc *ContainerImageUsecase) Exist(name string) (bool, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Exist(sock, name)
}

func (uc *ContainerImageUsecase) Pull(req *request.ContainerImagePull) error {
	sock := containerSock(uc.setting)
	return uc.repo.Pull(sock, req)
}

func (uc *ContainerImageUsecase) Remove(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(sock, id)
}

func (uc *ContainerImageUsecase) Prune() error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(sock)
}
