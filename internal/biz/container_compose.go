package biz

import "github.com/tnb-labs/panel/pkg/types"

type ContainerComposeRepo interface {
	List() ([]types.ContainerCompose, error)
	Get(name string) (string, string, error)
	Create(name, compose, env string) error
	Update(name, compose, env string) error
	Up(name string, force bool) error
	Down(name string) error
	Remove(name string) error
}
