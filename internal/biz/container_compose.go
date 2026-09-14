package biz

import (
	"context"

	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerComposeRepo interface {
	List(ctx context.Context) ([]types.ContainerCompose, error)
	Get(name string) (string, []types.KV, error)
	Create(name, compose string, envs []types.KV) error
	Update(name, compose string, envs []types.KV) error
	Up(ctx context.Context, name string, force bool) error
	Down(ctx context.Context, name string) error
	RemoveDir(name string) error
}

type ContainerComposeUsecase struct {
	repo ContainerComposeRepo
}

func NewContainerComposeUsecase(repo ContainerComposeRepo) *ContainerComposeUsecase {
	return &ContainerComposeUsecase{repo: repo}
}

func (uc *ContainerComposeUsecase) List(ctx context.Context) ([]types.ContainerCompose, error) {
	return uc.repo.List(ctx)
}

func (uc *ContainerComposeUsecase) Get(name string) (string, []types.KV, error) {
	return uc.repo.Get(name)
}

func (uc *ContainerComposeUsecase) Create(name, compose string, envs []types.KV) error {
	return uc.repo.Create(name, compose, envs)
}

func (uc *ContainerComposeUsecase) Update(name, compose string, envs []types.KV) error {
	return uc.repo.Update(name, compose, envs)
}

func (uc *ContainerComposeUsecase) Up(ctx context.Context, name string, force bool) error {
	return uc.repo.Up(ctx, name, force)
}

func (uc *ContainerComposeUsecase) Down(ctx context.Context, name string) error {
	return uc.repo.Down(ctx, name)
}

func (uc *ContainerComposeUsecase) Remove(ctx context.Context, name string) error {
	if err := uc.repo.Down(ctx, name); err != nil {
		return err
	}
	return uc.repo.RemoveDir(name)
}
