package biz

import (
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type ContainerVolumeRepo interface {
	List() ([]types.ContainerVolume, error)
	Create(req *request.ContainerVolumeCreate) (string, error)
	Remove(id string) error
	Prune() error
}
