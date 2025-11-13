package data

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/moby/moby/client"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/tools"
	"github.com/acepanel/panel/pkg/types"
)

type containerImageRepo struct{}

func NewContainerImageRepo() biz.ContainerImageRepo {
	return &containerImageRepo{}
}

// List 列出镜像
func (r *containerImageRepo) List() ([]types.ContainerImage, error) {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.ImageList(context.Background(), client.ImageListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var images []types.ContainerImage
	for _, item := range resp.Items {
		images = append(images, types.ContainerImage{
			ID:          item.ID,
			Containers:  item.Containers,
			RepoTags:    item.RepoTags,
			RepoDigests: item.RepoDigests,
			Size:        tools.FormatBytes(float64(item.Size)),
			Labels:      types.MapToKV(item.Labels),
			CreatedAt:   time.Unix(item.Created, 0),
		})
	}

	slices.SortFunc(images, func(a types.ContainerImage, b types.ContainerImage) int {
		return strings.Compare(a.ID, b.ID)
	})

	return images, nil
}

// Pull 拉取镜像
func (r *containerImageRepo) Pull(req *request.ContainerImagePull) error {
	var sb strings.Builder

	if req.Auth {
		sb.WriteString(fmt.Sprintf("docker login -u %s -p %s", req.Username, req.Password))
		if _, err := shell.Exec(sb.String()); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		sb.Reset()
	}

	sb.WriteString(fmt.Sprintf("docker pull %s", req.Name))

	if _, err := shell.Exec(sb.String()); err != nil {
		return err
	}

	return nil
}

// Remove 删除镜像
func (r *containerImageRepo) Remove(id string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "docker rmi %s", id)
	return err
}

// Prune 清理未使用的镜像
func (r *containerImageRepo) Prune() error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "docker image prune -f")
	return err
}
