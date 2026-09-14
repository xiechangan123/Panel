package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/types"
)

type containerImageRepo struct{}

func NewContainerImageRepo() biz.ContainerImageRepo {
	return &containerImageRepo{}
}

// List 列出镜像
func (r *containerImageRepo) List(ctx context.Context, sock string) ([]types.ContainerImage, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.ImageList(ctx, client.ImageListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	images := lo.Map(resp.Items, func(item image.Summary, _ int) types.ContainerImage {
		return types.ContainerImage{
			ID:          item.ID,
			Containers:  item.Containers,
			RepoTags:    item.RepoTags,
			RepoDigests: item.RepoDigests,
			Size:        tools.FormatBytes(float64(item.Size)),
			Labels:      types.MapToKV(item.Labels),
			CreatedAt:   time.Unix(item.Created, 0),
		}
	})

	slices.SortFunc(images, func(a types.ContainerImage, b types.ContainerImage) int {
		return strings.Compare(a.ID, b.ID)
	})

	return images, nil
}

// Exist 检查镜像是否存在
func (r *containerImageRepo) Exist(ctx context.Context, sock string, name string) (bool, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return false, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ImageInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Pull 拉取镜像
func (r *containerImageRepo) Pull(ctx context.Context, sock string, req *request.ContainerImagePull) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	options := client.ImagePullOptions{}
	if req.Auth {
		authConfig := registry.AuthConfig{
			Username: req.Username,
			Password: req.Password,
		}
		encodedJSON, err := json.Marshal(authConfig) //nolint:gosec
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		options.RegistryAuth = authStr
	}

	out, err := apiClient.ImagePull(ctx, req.Name, options)
	if err != nil {
		return err
	}
	defer func(out client.ImagePullResponse) { _ = out.Close() }(out)

	return out.Wait(ctx)
}

// Remove 删除镜像
func (r *containerImageRepo) Remove(ctx context.Context, sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	// 级联删除父层，中途取消会留下删了一半的镜像链
	_, err = apiClient.ImageRemove(context.WithoutCancel(ctx), id, client.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	return err
}

// Prune 清理未使用的镜像
func (r *containerImageRepo) Prune(ctx context.Context, sock string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	// 中途取消会留下清理到一半的状态
	_, err = apiClient.ImagePrune(context.WithoutCancel(ctx), client.ImagePruneOptions{
		Filters: make(client.Filters).
			Add("dangling", "false").
			Add("label", "created_by!=acepanel"),
	})
	return err
}
