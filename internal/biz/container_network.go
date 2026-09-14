package biz

import (
	"context"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerNetworkRepo interface {
	List(ctx context.Context, sock string) ([]types.ContainerNetwork, error)
	Create(ctx context.Context, sock string, req *request.ContainerNetworkCreate) (string, error)
	Remove(ctx context.Context, sock string, id string) error
	Prune(ctx context.Context, sock string) error
}

type ContainerNetworkUsecase struct {
	repo    ContainerNetworkRepo
	setting SettingRepo
}

func NewContainerNetworkUsecase(repo ContainerNetworkRepo, setting SettingRepo) *ContainerNetworkUsecase {
	return &ContainerNetworkUsecase{repo: repo, setting: setting}
}

func (uc *ContainerNetworkUsecase) List(ctx context.Context) ([]types.ContainerNetwork, error) {
	sock := containerSock(uc.setting)
	return uc.repo.List(ctx, sock)
}

func (uc *ContainerNetworkUsecase) Create(ctx context.Context, req *request.ContainerNetworkCreate) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Create(ctx, sock, req)
}

func (uc *ContainerNetworkUsecase) Remove(ctx context.Context, id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(ctx, sock, id)
}

func (uc *ContainerNetworkUsecase) Prune(ctx context.Context) error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(ctx, sock)
}
