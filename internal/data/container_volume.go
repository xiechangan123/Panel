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

type containerVolumeRepo struct{}

func NewContainerVolumeRepo() biz.ContainerVolumeRepo {
	return &containerVolumeRepo{}
}

// List 列出存储卷
func (r *containerVolumeRepo) List() ([]types.ContainerVolume, error) {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.VolumeList(context.Background(), client.VolumeListOptions{})
	if err != nil {
		return nil, err
	}

	var volumes []types.ContainerVolume
	for _, item := range resp.Items {
		createdAt, _ := time.Parse(time.RFC3339Nano, item.CreatedAt)
		volumes = append(volumes, types.ContainerVolume{
			Name:       item.Name,
			Driver:     item.Driver,
			Scope:      item.Scope,
			MountPoint: item.Mountpoint,
			CreatedAt:  createdAt,
			Labels:     types.MapToKV(item.Labels),
			Options:    types.MapToKV(item.Options),
			RefCount:   item.UsageData.RefCount,
			Size:       tools.FormatBytes(float64(item.UsageData.Size)),
		})
	}

	slices.SortFunc(volumes, func(a types.ContainerVolume, b types.ContainerVolume) int {
		return strings.Compare(a.Name, b.Name)
	})

	return volumes, nil
}

// Create 创建存储卷
func (r *containerVolumeRepo) Create(req *request.ContainerVolumeCreate) (string, error) {
	var sb strings.Builder
	sb.WriteString("docker volume create")
	sb.WriteString(fmt.Sprintf(" %s", req.Name))

	if req.Driver != "" {
		sb.WriteString(fmt.Sprintf(" --driver %s", req.Driver))
	}
	for _, label := range req.Labels {
		sb.WriteString(fmt.Sprintf(" --label %s=%s", label.Key, label.Value))
	}

	for _, option := range req.Options {
		sb.WriteString(fmt.Sprintf(" --opt %s=%s", option.Key, option.Value))
	}

	return shell.Exec(sb.String())
}

// Remove 删除存储卷
func (r *containerVolumeRepo) Remove(id string) error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "docker volume rm -f %s", id)
	return err
}

// Prune 清理未使用的存储卷
func (r *containerVolumeRepo) Prune() error {
	_, err := shell.ExecfWithTimeout(2*time.Minute, "docker volume prune -f")
	return err
}
