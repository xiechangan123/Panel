package biz

import (
	"context"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerVolumeRepo interface {
	List(ctx context.Context, sock string) ([]types.ContainerVolume, error)
	Create(ctx context.Context, sock string, req *request.ContainerVolumeCreate) (string, error)
	Remove(ctx context.Context, sock string, id string) error
	Prune(ctx context.Context, sock string) error
}

type ContainerVolumeUsecase struct {
	repo    ContainerVolumeRepo
	setting SettingRepo
}

func NewContainerVolumeUsecase(repo ContainerVolumeRepo, setting SettingRepo) *ContainerVolumeUsecase {
	return &ContainerVolumeUsecase{repo: repo, setting: setting}
}

func (uc *ContainerVolumeUsecase) List(ctx context.Context) ([]types.ContainerVolume, error) {
	sock := containerSock(uc.setting)
	return uc.repo.List(ctx, sock)
}

func (uc *ContainerVolumeUsecase) Create(ctx context.Context, req *request.ContainerVolumeCreate) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Create(ctx, sock, req)
}

func (uc *ContainerVolumeUsecase) Remove(ctx context.Context, id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(ctx, sock, id)
}

func (uc *ContainerVolumeUsecase) Prune(ctx context.Context) error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(ctx, sock)
}
