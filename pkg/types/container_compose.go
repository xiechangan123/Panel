package types

import "time"

// ContainerComposeRaw docker compose ls 命令原始输出
type ContainerComposeRaw struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

type ContainerCompose struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
