package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerRepo interface {
	ListAll() ([]types.Container, error)
	ListByName(name string) ([]types.Container, error)
	Create(req *request.ContainerCreate) (string, error)
	Remove(id string) error
	Start(id string) error
	Stop(id string) error
	Restart(id string) error
	Pause(id string) error
	Unpause(id string) error
	Kill(id string) error
	Rename(id string, newName string) error
	Logs(id string, tail int) (string, error)
	Prune() error
}

type ContainerUsecase struct {
	repo ContainerRepo
}

func NewContainerUsecase(repo ContainerRepo) *ContainerUsecase {
	return &ContainerUsecase{repo: repo}
}

func (uc *ContainerUsecase) ListAll() ([]types.Container, error) {
	return uc.repo.ListAll()
}

func (uc *ContainerUsecase) ListByName(name string) ([]types.Container, error) {
	return uc.repo.ListByName(name)
}

func (uc *ContainerUsecase) Create(req *request.ContainerCreate) (string, error) {
	return uc.repo.Create(req)
}

func (uc *ContainerUsecase) Remove(id string) error {
	return uc.repo.Remove(id)
}

func (uc *ContainerUsecase) Start(id string) error {
	return uc.repo.Start(id)
}

func (uc *ContainerUsecase) Stop(id string) error {
	return uc.repo.Stop(id)
}

func (uc *ContainerUsecase) Restart(id string) error {
	return uc.repo.Restart(id)
}

func (uc *ContainerUsecase) Pause(id string) error {
	return uc.repo.Pause(id)
}

func (uc *ContainerUsecase) Unpause(id string) error {
	return uc.repo.Unpause(id)
}

func (uc *ContainerUsecase) Kill(id string) error {
	return uc.repo.Kill(id)
}

func (uc *ContainerUsecase) Rename(id string, newName string) error {
	return uc.repo.Rename(id, newName)
}

func (uc *ContainerUsecase) Logs(id string, tail int) (string, error) {
	return uc.repo.Logs(id, tail)
}

func (uc *ContainerUsecase) Prune() error {
	return uc.repo.Prune()
}
