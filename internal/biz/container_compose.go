package biz

type ContainerComposeRepo interface {
	List() ([]string, error)
	Get(name string) (string, string, error)
	Create(name, compose, env string) error
	Up(name string, force bool) error
	Down(name string) error
	Remove(name string) error
}
