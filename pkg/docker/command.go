package docker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/libtnb/utils/str"

	"github.com/acepanel/panel/v3/internal/request"
)

// ImagePullShell 生成拉取镜像的命令，需要认证时返回配套的清理命令
func ImagePullShell(sock string, req *request.ContainerImagePull) (string, string, error) {
	if !req.Auth {
		return Command(sock, "pull", req.Name), "", nil
	}

	named, err := reference.ParseNormalizedNamed(req.Name)
	if err != nil {
		return "", "", fmt.Errorf("invalid image reference: %w", err)
	}

	configDir := filepath.Join(os.TempDir(), "ace-docker-task-"+str.Random(16))
	cleanup := "rm -rf " + shellQuote(configDir)
	shell := strings.Join([]string{
		"set -e",
		"mkdir -p " + shellQuote(configDir),
		"trap " + shellQuote(cleanup) + " EXIT",
		"printf %s " + shellQuote(req.Password) + " | " + Command(sock, "--config", configDir, "login", "--username", req.Username, "--password-stdin", reference.Domain(named)),
		Command(sock, "--config", configDir, "pull", req.Name),
	}, "\n")

	return shell, cleanup, nil
}

// RunShell 生成后台创建容器的命令
func RunShell(sock string, req *request.ContainerCreate) (string, error) {
	args := []string{"run", "--detach"}

	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}
	if req.Network != "" {
		args = append(args, "--network", req.Network)
	}
	if req.Hostname != "" {
		args = append(args, "--hostname", req.Hostname)
	}
	if req.WorkingDir != "" {
		args = append(args, "--workdir", req.WorkingDir)
	}
	if req.User != "" {
		args = append(args, "--user", req.User)
	}
	for _, alias := range req.NetworkAliases {
		args = append(args, "--network-alias", alias)
	}
	if req.StaticIP != "" {
		args = append(args, "--ip", req.StaticIP)
	}
	for _, dns := range req.DNS {
		args = append(args, "--dns", dns)
	}
	for _, host := range req.ExtraHosts {
		args = append(args, "--add-host", host)
	}
	for _, capability := range req.CapAdd {
		args = append(args, "--cap-add", capability)
	}
	for _, capability := range req.CapDrop {
		args = append(args, "--cap-drop", capability)
	}
	for _, device := range req.Devices {
		args = append(args, "--device", mountSpec(device.Host, device.Container, device.Permissions))
	}
	for _, option := range req.SecurityOpt {
		args = append(args, "--security-opt", option)
	}
	for _, sysctl := range req.Sysctls {
		args = append(args, "--sysctl", sysctl.Key+"="+sysctl.Value)
	}
	for _, ulimit := range req.Ulimits {
		args = append(args, "--ulimit", fmt.Sprintf("%s=%d:%d", ulimit.Name, ulimit.Soft, ulimit.Hard))
	}
	for _, tmpfs := range req.Tmpfs {
		value := tmpfs.Key
		if tmpfs.Value != "" {
			value += ":" + tmpfs.Value
		}
		args = append(args, "--tmpfs", value)
	}
	if req.ShmSize > 0 {
		args = append(args, "--shm-size", strconv.FormatInt(req.ShmSize, 10))
	}
	if req.Init {
		args = append(args, "--init")
	}
	if req.StopSignal != "" {
		args = append(args, "--stop-signal", req.StopSignal)
	}
	if req.StopTimeout > 0 {
		args = append(args, "--stop-timeout", strconv.Itoa(req.StopTimeout))
	}
	if req.ReadonlyRootfs {
		args = append(args, "--read-only")
	}
	if req.Healthcheck != nil && len(req.Healthcheck.Test) > 0 {
		test := req.Healthcheck.Test
		if test[0] == "CMD" || test[0] == "CMD-SHELL" {
			test = test[1:]
		}
		args = append(args, "--health-cmd", strings.Join(test, " "))
		if req.Healthcheck.Interval > 0 {
			args = append(args, "--health-interval", req.Healthcheck.Interval.String())
		}
		if req.Healthcheck.Timeout > 0 {
			args = append(args, "--health-timeout", req.Healthcheck.Timeout.String())
		}
		if req.Healthcheck.StartPeriod > 0 {
			args = append(args, "--health-start-period", req.Healthcheck.StartPeriod.String())
		}
		if req.Healthcheck.Retries > 0 {
			args = append(args, "--health-retries", strconv.Itoa(req.Healthcheck.Retries))
		}
	}
	if req.PublishAllPorts {
		args = append(args, "--publish-all")
	} else {
		for _, port := range req.Ports {
			if port.ContainerStart < 1 || port.ContainerEnd > 65535 || port.ContainerStart > port.ContainerEnd ||
				port.HostStart < 1 || port.HostEnd > 65535 || port.HostStart > port.HostEnd {
				return "", errors.New("port range is invalid")
			}
			if port.ContainerEnd-port.ContainerStart != port.HostEnd-port.HostStart {
				return "", errors.New("container port and host port count do not match")
			}

			for offset := uint(0); offset <= port.HostEnd-port.HostStart; offset++ {
				host := ""
				if port.Host.IsValid() {
					host = port.Host.String() + ":"
					if port.Host.Is6() {
						host = "[" + port.Host.String() + "]:"
					}
				}
				mapping := fmt.Sprintf("%s%d:%d/%s", host, port.HostStart+offset, port.ContainerStart+offset, port.Protocol)
				args = append(args, "--publish", mapping)
			}
		}
	}

	for _, volume := range req.Volumes {
		args = append(args, "--volume", mountSpec(volume.Host, volume.Container, volume.Mode))
	}
	for _, env := range req.Env {
		args = append(args, "--env", env.Key+"="+env.Value)
	}
	for _, label := range req.Labels {
		args = append(args, "--label", label.Key+"="+label.Value)
	}

	if len(req.Entrypoint) > 0 {
		args = append(args, "--entrypoint", req.Entrypoint[0])
	}
	if req.RestartPolicy != "" {
		restartPolicy := req.RestartPolicy
		if restartPolicy == "on-failure" {
			restartPolicy += ":5"
		}
		args = append(args, "--restart", restartPolicy)
	}
	if req.AutoRemove {
		args = append(args, "--rm")
	}
	if req.Privileged {
		args = append(args, "--privileged")
	}
	if req.OpenStdin {
		args = append(args, "--interactive")
	}
	if req.Tty {
		args = append(args, "--tty")
	}
	if req.CPUShares > 0 {
		args = append(args, "--cpu-shares", strconv.FormatInt(req.CPUShares, 10))
	}
	if req.CPUs > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(req.CPUs, 'f', -1, 64))
	}
	if req.Memory > 0 {
		args = append(args, "--memory", strconv.FormatInt(req.Memory, 10)+"m")
	}

	args = append(args, req.Image)
	if len(req.Entrypoint) > 1 {
		args = append(args, req.Entrypoint[1:]...)
	}
	args = append(args, req.Command...)

	return Command(sock, args...), nil
}

// mountSpec 拼接 docker 的 源:目标[:选项] 形式，
// 选项为空时不能留下尾随冒号，否则 docker 报 empty section between colons
func mountSpec(source, target, option string) string {
	spec := source + ":" + target
	if option != "" {
		spec += ":" + option
	}
	return spec
}

// Command 拼接指向指定套接字的 docker 命令，各参数均做转义
func Command(sock string, args ...string) string {
	args = append([]string{"docker", "--host", sock}, args...)
	for i := range args {
		args[i] = shellQuote(args[i])
	}
	return strings.Join(args, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
