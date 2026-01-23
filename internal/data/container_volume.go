package data

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/moby/moby/client"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
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
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return "", err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.VolumeCreate(context.Background(), client.VolumeCreateOptions{
		Name:       req.Name,
		Driver:     req.Driver,
		DriverOpts: types.KVToMap(req.Options),
		Labels:     types.KVToMap(req.Labels),
	})
	if err != nil {
		return "", err
	}

	return resp.Volume.Name, nil
}

// Remove 删除存储卷
func (r *containerVolumeRepo) Remove(id string) error {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.VolumeRemove(context.Background(), id, client.VolumeRemoveOptions{
		Force: true,
	})
	return err
}

// Prune 清理未使用的存储卷
func (r *containerVolumeRepo) Prune() error {
	apiClient, err := getDockerClient("/var/run/docker.sock")
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.VolumePrune(context.Background(), client.VolumePruneOptions{
		Filters: make(client.Filters).Add("label", "created_by!=acepanel"),
	})
	return err
}
