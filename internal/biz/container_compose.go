package biz

import "github.com/acepanel/panel/v3/pkg/types"

type ContainerComposeRepo interface {
	List() ([]types.ContainerCompose, error)
	Get(name string) (string, []types.KV, error)
	Create(name, compose string, envs []types.KV) error
	Update(name, compose string, envs []types.KV) error
	Up(name string, force bool) error
	Down(name string) error
	RemoveDir(name string) error
}

type ContainerComposeUsecase struct {
	repo ContainerComposeRepo
}

func NewContainerComposeUsecase(repo ContainerComposeRepo) *ContainerComposeUsecase {
	return &ContainerComposeUsecase{repo: repo}
}

func (uc *ContainerComposeUsecase) List() ([]types.ContainerCompose, error) {
	return uc.repo.List()
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

func (uc *ContainerComposeUsecase) Up(name string, force bool) error {
	return uc.repo.Up(name, force)
}

func (uc *ContainerComposeUsecase) Down(name string) error {
	return uc.repo.Down(name)
}

func (uc *ContainerComposeUsecase) Remove(name string) error {
	if err := uc.repo.Down(name); err != nil {
		return err
	}
	return uc.repo.RemoveDir(name)
}
