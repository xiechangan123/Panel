package biz

type ContainerComposeRepo interface {
	List() ([]string, error)
	Get(name string) (string, error)
	Create(name, compose string) error
	Up(name string, force bool) error
	Down(name string) error
	Remove(name string) error
}
