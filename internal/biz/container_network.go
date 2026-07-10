package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerNetworkRepo interface {
	List(sock string) ([]types.ContainerNetwork, error)
	Create(sock string, req *request.ContainerNetworkCreate) (string, error)
	Remove(sock string, id string) error
	Prune(sock string) error
}

type ContainerNetworkUsecase struct {
	repo    ContainerNetworkRepo
	setting SettingRepo
}

func NewContainerNetworkUsecase(repo ContainerNetworkRepo, setting SettingRepo) *ContainerNetworkUsecase {
	return &ContainerNetworkUsecase{repo: repo, setting: setting}
}

func (uc *ContainerNetworkUsecase) List() ([]types.ContainerNetwork, error) {
	sock := containerSock(uc.setting)
	return uc.repo.List(sock)
}

func (uc *ContainerNetworkUsecase) Create(req *request.ContainerNetworkCreate) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Create(sock, req)
}

func (uc *ContainerNetworkUsecase) Remove(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(sock, id)
}

func (uc *ContainerNetworkUsecase) Prune() error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(sock)
}
