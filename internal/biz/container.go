package biz

import (
	"slices"
	"strings"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type ContainerRepo interface {
	ListAll(sock string) ([]types.Container, error)
	Inspect(sock string, id string) (any, error)
	Create(sock string, req *request.ContainerCreate) (string, error)
	Remove(sock string, id string) error
	Start(sock string, id string) error
	Stop(sock string, id string) error
	Restart(sock string, id string) error
	Pause(sock string, id string) error
	Unpause(sock string, id string) error
	Kill(sock string, id string) error
	Rename(sock string, id string, newName string) error
	Logs(sock string, id string, tail int) (string, error)
	Prune(sock string) error
}

type ContainerUsecase struct {
	repo    ContainerRepo
	setting SettingRepo
}

func NewContainerUsecase(repo ContainerRepo, setting SettingRepo) *ContainerUsecase {
	return &ContainerUsecase{repo: repo, setting: setting}
}

func (uc *ContainerUsecase) ListAll() ([]types.Container, error) {
	sock := containerSock(uc.setting)
	return uc.repo.ListAll(sock)
}

func (uc *ContainerUsecase) ListByName(name string) ([]types.Container, error) {
	sock := containerSock(uc.setting)
	containers, err := uc.repo.ListAll(sock)
	if err != nil {
		return nil, err
	}

	containers = slices.DeleteFunc(containers, func(item types.Container) bool {
		return !strings.Contains(item.Name, name)
	})

	return containers, nil
}

func (uc *ContainerUsecase) Inspect(id string) (any, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Inspect(sock, id)
}

func (uc *ContainerUsecase) Create(req *request.ContainerCreate) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Create(sock, req)
}

// Update 删除旧容器后按新配置重建同名容器
func (uc *ContainerUsecase) Update(id string, req *request.ContainerCreate) (string, error) {
	sock := containerSock(uc.setting)
	if err := uc.repo.Remove(sock, id); err != nil {
		return "", err
	}
	return uc.repo.Create(sock, req)
}

func (uc *ContainerUsecase) Remove(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(sock, id)
}

func (uc *ContainerUsecase) Start(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Start(sock, id)
}

func (uc *ContainerUsecase) Stop(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Stop(sock, id)
}

func (uc *ContainerUsecase) Restart(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Restart(sock, id)
}

func (uc *ContainerUsecase) Pause(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Pause(sock, id)
}

func (uc *ContainerUsecase) Unpause(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Unpause(sock, id)
}

func (uc *ContainerUsecase) Kill(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Kill(sock, id)
}

func (uc *ContainerUsecase) Rename(id string, newName string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Rename(sock, id, newName)
}

func (uc *ContainerUsecase) Logs(id string, tail int) (string, error) {
	sock := containerSock(uc.setting)
	return uc.repo.Logs(sock, id, tail)
}

func (uc *ContainerUsecase) Prune() error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(sock)
}
