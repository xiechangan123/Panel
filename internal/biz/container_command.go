package biz

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

func containerImagePullShell(sock string, req *request.ContainerImagePull) (string, string, error) {
	if !req.Auth {
		return dockerCommand(sock, "pull", req.Name), "", nil
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
		"printf %s " + shellQuote(req.Password) + " | " + dockerCommand(sock, "--config", configDir, "login", "--username", req.Username, "--password-stdin", reference.Domain(named)),
		dockerCommand(sock, "--config", configDir, "pull", req.Name),
	}, "\n")

	return shell, cleanup, nil
}

func containerRunShell(sock string, req *request.ContainerCreate) (string, error) {
	args := []string{"run", "--detach"}

	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}
	if req.Network != "" {
		args = append(args, "--network", req.Network)
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
		args = append(args, "--volume", strings.Join([]string{volume.Host, volume.Container, volume.Mode}, ":"))
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

	return dockerCommand(sock, args...), nil
}

func dockerCommand(sock string, args ...string) string {
	args = append([]string{"docker", "--host", sock}, args...)
	for i := range args {
		args[i] = shellQuote(args[i])
	}
	return strings.Join(args, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
