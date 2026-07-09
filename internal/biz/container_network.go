package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerNetworkRepo interface {
	List() ([]types.ContainerNetwork, error)
	Create(req *request.ContainerNetworkCreate) (string, error)
	Remove(id string) error
	Prune() error
}

type ContainerNetworkUsecase struct {
	repo ContainerNetworkRepo
}

func NewContainerNetworkUsecase(repo ContainerNetworkRepo) *ContainerNetworkUsecase {
	return &ContainerNetworkUsecase{repo: repo}
}

func (uc *ContainerNetworkUsecase) List() ([]types.ContainerNetwork, error) {
	return uc.repo.List()
}

func (uc *ContainerNetworkUsecase) Create(req *request.ContainerNetworkCreate) (string, error) {
	return uc.repo.Create(req)
}

func (uc *ContainerNetworkUsecase) Remove(id string) error {
	return uc.repo.Remove(id)
}

func (uc *ContainerNetworkUsecase) Prune() error {
	return uc.repo.Prune()
}
