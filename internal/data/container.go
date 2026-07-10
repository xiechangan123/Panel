package data

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/docker"
	"github.com/acepanel/panel/v3/pkg/types"
)

type containerRepo struct{}

func NewContainerRepo(i do.Injector) (biz.ContainerRepo, error) {
	return &containerRepo{}, nil
}

// ListAll 列出所有容器
func (r *containerRepo) ListAll(sock string) ([]types.Container, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.ContainerList(context.Background(), client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	var containers []types.Container
	for _, item := range resp.Items {
		ports := make([]types.ContainerPort, 0)
		for _, port := range item.Ports {
			ports = append(ports, types.ContainerPort{
				ContainerStart: uint(port.PrivatePort),
				ContainerEnd:   uint(port.PrivatePort),
				HostStart:      uint(port.PublicPort),
				HostEnd:        uint(port.PublicPort),
				Protocol:       port.Type,
				Host:           port.IP,
			})
		}
		if len(item.Names) == 0 {
			item.Names = append(item.Names, "")
		}
		containers = append(containers, types.Container{
			ID:        item.ID,
			Name:      strings.TrimPrefix(item.Names[0], "/"), // https://github.com/moby/moby/issues/7519
			Image:     item.Image,
			ImageID:   item.ImageID,
			Command:   item.Command,
			CreatedAt: time.Unix(item.Created, 0),
			State:     string(item.State),
			Status:    item.Status,
			Ports:     ports,
			Labels:    types.MapToKV(item.Labels),
		})
	}

	slices.SortFunc(containers, func(a types.Container, b types.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	return containers, nil
}

// Create 创建容器
func (r *containerRepo) Create(sock string, req *request.ContainerCreate) (string, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return "", err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	ctx := context.Background()

	// 获取镜像信息
	image, err := apiClient.ImageInspect(ctx, req.Image)
	if err != nil {
		return "", fmt.Errorf("failed to inspect image: %v", err)
	}
	// 兼容一些没有指定命令和入口点的镜像
	if image.Config != nil {
		if len(req.Command) == 0 && len(image.Config.Cmd) > 0 {
			req.Command = image.Config.Cmd
		}
		if len(req.Entrypoint) == 0 && len(image.Config.Entrypoint) > 0 {
			req.Entrypoint = image.Config.Entrypoint
		}
	}

	// 构建容器配置
	config := &container.Config{
		Image:        req.Image,
		Tty:          req.Tty,
		OpenStdin:    req.OpenStdin,
		AttachStdin:  req.OpenStdin,
		AttachStdout: true,
		AttachStderr: true,
		Env:          types.KVToSlice(req.Env),
		Labels:       types.KVToMap(req.Labels),
		Entrypoint:   req.Entrypoint,
		Cmd:          req.Command,
	}

	// 构建主机配置
	hostConfig := &container.HostConfig{
		AutoRemove:      req.AutoRemove,
		Privileged:      req.Privileged,
		PublishAllPorts: req.PublishAllPorts,
	}

	// 构建网络配置
	networkConfig := &network.NetworkingConfig{}
	if req.Network != "" {
		switch req.Network {
		case "host", "none", "bridge":
			hostConfig.NetworkMode = container.NetworkMode(req.Network)
		}
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{req.Network: {}}
	}

	// 设置端口映射
	if !req.PublishAllPorts && len(req.Ports) > 0 {
		portMap := make(network.PortMap)
		for _, port := range req.Ports {
			if port.ContainerStart-port.ContainerEnd != port.HostStart-port.HostEnd {
				return "", fmt.Errorf("container port and host port count do not match (container: %d host: %d)", port.ContainerStart-port.ContainerEnd, port.HostStart-port.HostEnd)
			}
			if port.ContainerStart > port.ContainerEnd || port.HostStart > port.HostEnd || port.ContainerStart < 1 || port.HostStart < 1 {
				return "", fmt.Errorf("port range is invalid")
			}

			count := uint(0)
			for i := port.HostStart; i <= port.HostEnd; i++ {
				bindItem := network.PortBinding{HostIP: port.Host, HostPort: strconv.Itoa(int(i))}
				portMap[network.MustParsePort(fmt.Sprintf("%d/%s", port.ContainerStart+count, port.Protocol))] = []network.PortBinding{bindItem}
				count++
			}
		}

		exposed := make(network.PortSet)
		for port := range portMap {
			exposed[port] = struct{}{}
		}

		config.ExposedPorts = exposed
		hostConfig.PortBindings = portMap
	}
	// 设置卷挂载
	volumes := make(map[string]struct{})
	for _, v := range req.Volumes {
		volumes[v.Container] = struct{}{}
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s:%s", v.Host, v.Container, v.Mode))
	}
	config.Volumes = volumes
	// 设置重启策略
	hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.RestartPolicy)}
	if req.RestartPolicy == "on-failure" {
		hostConfig.RestartPolicy.MaximumRetryCount = 5
	}
	// 设置资源限制
	hostConfig.CPUShares = req.CPUShares
	hostConfig.NanoCPUs = req.CPUs * 1e9
	hostConfig.Memory = req.Memory * 1024 * 1024
	hostConfig.MemorySwap = 0

	// 创建容器
	resp, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:             req.Name,
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkConfig,
	})
	if err != nil {
		return "", err
	}

	// 启动容器
	_, _ = apiClient.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	return resp.ID, nil
}

// Remove 移除容器
func (r *containerRepo) Remove(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerRemove(context.Background(), id, client.ContainerRemoveOptions{
		Force: true,
	})
	return err
}

// Start 启动容器
func (r *containerRepo) Start(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerStart(context.Background(), id, client.ContainerStartOptions{})
	return err
}

// Stop 停止容器
func (r *containerRepo) Stop(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerStop(context.Background(), id, client.ContainerStopOptions{})
	return err
}

// Restart 重启容器
func (r *containerRepo) Restart(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerRestart(context.Background(), id, client.ContainerRestartOptions{})
	return err
}

// Pause 暂停容器
func (r *containerRepo) Pause(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerPause(context.Background(), id, client.ContainerPauseOptions{})
	return err
}

// Unpause 恢复容器
func (r *containerRepo) Unpause(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerUnpause(context.Background(), id, client.ContainerUnpauseOptions{})
	return err
}

// Kill 杀死容器
func (r *containerRepo) Kill(sock string, id string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerKill(context.Background(), id, client.ContainerKillOptions{
		Signal: "KILL",
	})
	return err
}

// Rename 重命名容器
func (r *containerRepo) Rename(sock string, id string, newName string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerRename(context.Background(), id, client.ContainerRenameOptions{
		NewName: newName,
	})
	return err
}

// Logs 查看容器末尾 tail 行日志
func (r *containerRepo) Logs(sock string, id string, tail int) (string, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return "", err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	// 非 TTY 容器日志为多路复用流，需按 TTY 设置决定是否解复用
	inspect, err := apiClient.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
	if err != nil {
		return "", err
	}
	tty := inspect.Container.Config != nil && inspect.Container.Config.Tty

	reader, err := apiClient.ContainerLogs(context.Background(), id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	})
	if err != nil {
		return "", err
	}
	defer func(reader client.ContainerLogsResult) { _ = reader.Close() }(reader)

	var buf bytes.Buffer
	if err = docker.CopyLogs(&buf, reader, tty); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Prune 清理未使用的容器
func (r *containerRepo) Prune(sock string) error {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	_, err = apiClient.ContainerPrune(context.Background(), client.ContainerPruneOptions{
		Filters: make(client.Filters).Add("label", "created_by!=acepanel"),
	})
	return err
}
