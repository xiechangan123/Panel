package biz

import (
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/docker"
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
	task    TaskRepo
	t       *gotext.Locale
}

func NewContainerImageUsecase(t *gotext.Locale, containerImageRepo ContainerImageRepo, settingRepo SettingRepo, taskRepo TaskRepo) *ContainerImageUsecase {
	return &ContainerImageUsecase{
		repo:    containerImageRepo,
		setting: settingRepo,
		task:    taskRepo,
		t:       t,
	}
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

func (uc *ContainerImageUsecase) PullBackground(req *request.ContainerImagePull) error {
	shell, cancelShell, err := docker.ImagePullShell(containerSock(uc.setting), req)
	if err != nil {
		return err
	}

	task := new(Task)
	task.Key = "container:image:pull:" + req.Name
	task.Name = uc.t.Get("Pull image %s", req.Name)
	task.Status = TaskStatusWaiting
	task.Shell = shell
	task.CancelShell = cancelShell

	return uc.task.Push(task)
}

func (uc *ContainerImageUsecase) Remove(id string) error {
	sock := containerSock(uc.setting)
	return uc.repo.Remove(sock, id)
}

func (uc *ContainerImageUsecase) Prune() error {
	sock := containerSock(uc.setting)
	return uc.repo.Prune(sock)
}
