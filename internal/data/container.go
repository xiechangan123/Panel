package data

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/docker"
	"github.com/acepanel/panel/v3/pkg/types"
)

type containerRepo struct{}

func NewContainerRepo() biz.ContainerRepo {
	return &containerRepo{}
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
		ports := make([]types.ContainerPort, len(item.Ports))
		for i, port := range item.Ports {
			ports[i] = types.ContainerPort{
				ContainerStart: uint(port.PrivatePort),
				ContainerEnd:   uint(port.PrivatePort),
				HostStart:      uint(port.PublicPort),
				HostEnd:        uint(port.PublicPort),
				Protocol:       port.Type,
				Host:           port.IP,
			}
		}
		slices.SortFunc(ports, func(a, b types.ContainerPort) int {
			aOffset := int64(a.HostStart) - int64(a.ContainerStart)
			if a.HostStart == 0 {
				aOffset = 0
			}
			bOffset := int64(b.HostStart) - int64(b.ContainerStart)
			if b.HostStart == 0 {
				bOffset = 0
			}
			return cmp.Or(
				a.Host.Compare(b.Host),
				strings.Compare(a.Protocol, b.Protocol),
				cmp.Compare(aOffset, bOffset),
				cmp.Compare(a.ContainerStart, b.ContainerStart),
				cmp.Compare(a.HostStart, b.HostStart),
			)
		})

		merged := ports[:0]
		for _, port := range ports {
			if len(merged) > 0 {
				last := &merged[len(merged)-1]
				if last.Host == port.Host && last.Protocol == port.Protocol &&
					last.ContainerEnd+1 == port.ContainerStart && last.HostEnd+1 == port.HostStart {
					last.ContainerEnd = port.ContainerEnd
					last.HostEnd = port.HostEnd
					continue
				}
			}
			merged = append(merged, port)
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
			Ports:     merged,
			Labels:    types.MapToKV(item.Labels),
		})
	}

	slices.SortFunc(containers, func(a types.Container, b types.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	return containers, nil
}

// Inspect 获取容器详细信息
func (r *containerRepo) Inspect(sock string, id string) (any, error) {
	apiClient, err := getDockerClient(sock)
	if err != nil {
		return nil, err
	}
	defer func(apiClient *client.Client) { _ = apiClient.Close() }(apiClient)

	resp, err := apiClient.ContainerInspect(context.Background(), id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}

	return resp.Container, nil
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
		Hostname:     req.Hostname,
		WorkingDir:   req.WorkingDir,
		User:         req.User,
		Tty:          req.Tty,
		OpenStdin:    req.OpenStdin,
		AttachStdin:  req.OpenStdin,
		AttachStdout: true,
		AttachStderr: true,
		Env:          types.KVToSlice(req.Env),
		Labels:       types.KVToMap(req.Labels),
		Entrypoint:   req.Entrypoint,
		Cmd:          req.Command,
		StopSignal:   req.StopSignal,
	}
	if req.StopTimeout > 0 {
		config.StopTimeout = &req.StopTimeout
	}
	if req.Healthcheck != nil {
		config.Healthcheck = &container.HealthConfig{
			Test: req.Healthcheck.Test, Interval: req.Healthcheck.Interval, Timeout: req.Healthcheck.Timeout,
			StartPeriod: req.Healthcheck.StartPeriod, Retries: req.Healthcheck.Retries,
		}
	}

	// 构建主机配置
	hostConfig := &container.HostConfig{
		AutoRemove:      req.AutoRemove,
		Privileged:      req.Privileged,
		PublishAllPorts: req.PublishAllPorts,
		ReadonlyRootfs:  req.ReadonlyRootfs,
		ExtraHosts:      req.ExtraHosts,
		CapAdd:          req.CapAdd,
		CapDrop:         req.CapDrop,
		SecurityOpt:     req.SecurityOpt,
		Sysctls:         types.KVToMap(req.Sysctls),
		Tmpfs:           types.KVToMap(req.Tmpfs),
		ShmSize:         req.ShmSize,
	}
	if req.Init {
		hostConfig.Init = &req.Init
	}
	for _, dns := range req.DNS {
		if address, parseErr := netip.ParseAddr(dns); parseErr == nil {
			hostConfig.DNS = append(hostConfig.DNS, address)
		}
	}
	for _, device := range req.Devices {
		hostConfig.Devices = append(hostConfig.Devices, container.DeviceMapping{
			PathOnHost: device.Host, PathInContainer: device.Container, CgroupPermissions: device.Permissions,
		})
	}
	for _, ulimit := range req.Ulimits {
		hostConfig.Ulimits = append(hostConfig.Ulimits, &container.Ulimit{Name: ulimit.Name, Soft: ulimit.Soft, Hard: ulimit.Hard})
	}

	// 构建网络配置
	networkConfig := &network.NetworkingConfig{}
	if req.Network != "" {
		switch req.Network {
		case "host", "none":
			hostConfig.NetworkMode = container.NetworkMode(req.Network)
		case "bridge":
			hostConfig.NetworkMode = container.NetworkMode(req.Network)
		default:
			endpoint := &network.EndpointSettings{Aliases: req.NetworkAliases}
			if req.StaticIP != "" {
				address, parseErr := netip.ParseAddr(req.StaticIP)
				if parseErr != nil {
					return "", fmt.Errorf("invalid static IP address: %w", parseErr)
				}
				endpoint.IPAddress = address
			}
			networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{req.Network: endpoint}
		}
	}

	// 设置端口映射
	if !req.PublishAllPorts && len(req.Ports) > 0 {
		portMap := make(network.PortMap)
		for _, port := range req.Ports {
			if port.ContainerStart-port.ContainerEnd != port.HostStart-port.HostEnd {
				return "", fmt.Errorf("container port and host port count do not match (container: %d host: %d)", port.ContainerStart-port.ContainerEnd, port.HostStart-port.HostEnd)
			}
			if port.ContainerStart > port.ContainerEnd || port.HostStart > port.HostEnd || port.ContainerStart < 1 || port.HostStart < 1 {
				return "", errors.New("port range is invalid")
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
	hostConfig.NanoCPUs = int64(req.CPUs * 1e9)
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
