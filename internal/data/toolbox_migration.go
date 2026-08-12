package data

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.yaml.in/yaml/v4"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type toolboxMigrationSourceRepo struct {
	t *gotext.Locale
}

type toolboxMigrationAdapter interface {
	Probe(ctx context.Context) (*types.MigrationSourceInfo, error)
	Items(ctx context.Context) ([]types.MigrationSourceItem, error)
	Detail(ctx context.Context, item types.MigrationSourceItem) (*types.MigrationSourceDetail, error)
	SetRunning(ctx context.Context, detail *types.MigrationSourceDetail, running bool) error
	Prepare(ctx context.Context, detail *types.MigrationSourceDetail) (*types.MigrationArtifact, error)
	Download(ctx context.Context, artifact *types.MigrationArtifact, target string) error
}

func NewToolboxMigrationSourceRepo(t *gotext.Locale) (biz.ToolboxMigrationSourceRepo, error) {
	return &toolboxMigrationSourceRepo{t: t}, nil
}

func (r *toolboxMigrationSourceRepo) adapter(conn *request.ToolboxMigrationConnection) (toolboxMigrationAdapter, error) {
	switch conn.SourcePanel {
	case "baota":
		return &baotaMigrationAdapter{source: r, conn: conn}, nil
	case "onepanel":
		return &onePanelMigrationAdapter{source: r, conn: conn}, nil
	default:
		return nil, errors.New("unsupported source panel")
	}
}

func (r *toolboxMigrationSourceRepo) Probe(ctx context.Context, conn *request.ToolboxMigrationConnection) (*types.MigrationSourceInfo, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Probe(ctx)
}

func (r *toolboxMigrationSourceRepo) Items(ctx context.Context, conn *request.ToolboxMigrationConnection) ([]types.MigrationSourceItem, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Items(ctx)
}

func (r *toolboxMigrationSourceRepo) Detail(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	item types.MigrationSourceItem,
) (*types.MigrationSourceDetail, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Detail(ctx, item)
}

func (r *toolboxMigrationSourceRepo) parseContainerDetail(raw any, item *types.MigrationSourceItem) *types.MigrationContainerDetail {
	if encoded, ok := raw.(string); ok {
		_ = json.Unmarshal([]byte(encoded), &raw)
	}
	info := cast.ToStringMap(raw)
	config := cast.ToStringMap(info["Config"])
	hostConfig := cast.ToStringMap(info["HostConfig"])
	detail := &types.MigrationContainerDetail{
		ID: item.SourceID, Image: cast.ToString(config["Image"]),
		Entrypoint: r.strings(config["Entrypoint"]), Command: r.strings(config["Cmd"]),
		Env: r.environment(config["Env"]), Labels: types.MapToKV(cast.ToStringMapString(config["Labels"])),
		RestartPolicy: cast.ToString(cast.ToStringMap(hostConfig["RestartPolicy"])["Name"]),
		Hostname:      cast.ToString(config["Hostname"]), WorkingDir: cast.ToString(config["WorkingDir"]), User: cast.ToString(config["User"]),
		Privileged: cast.ToBool(hostConfig["Privileged"]), AutoRemove: cast.ToBool(hostConfig["AutoRemove"]),
		ReadonlyRootfs: cast.ToBool(hostConfig["ReadonlyRootfs"]), OpenStdin: cast.ToBool(config["OpenStdin"]),
		PublishAllPorts: cast.ToBool(hostConfig["PublishAllPorts"]), Tty: cast.ToBool(config["Tty"]),
		CPUShares: cast.ToInt64(hostConfig["CpuShares"]), CPUs: float64(cast.ToInt64(hostConfig["NanoCpus"])) / 1e9,
		Memory: cast.ToInt64(hostConfig["Memory"]) / 1024 / 1024,
		DNS:    r.strings(hostConfig["Dns"]), ExtraHosts: r.strings(hostConfig["ExtraHosts"]),
		CapAdd: r.strings(hostConfig["CapAdd"]), CapDrop: r.strings(hostConfig["CapDrop"]),
		SecurityOpt: r.strings(hostConfig["SecurityOpt"]), Sysctls: types.MapToKV(cast.ToStringMapString(hostConfig["Sysctls"])),
		Tmpfs:   types.MapToKV(cast.ToStringMapString(hostConfig["Tmpfs"])),
		ShmSize: cast.ToInt64(hostConfig["ShmSize"]), Init: cast.ToBool(hostConfig["Init"]),
		StopSignal: cast.ToString(config["StopSignal"]), StopTimeout: int(cast.ToInt64(config["StopTimeout"])),
		Running: item.Status == "running",
	}
	for _, rawDevice := range cast.ToSlice(hostConfig["Devices"]) {
		device := cast.ToStringMap(rawDevice)
		detail.Devices = append(detail.Devices, types.ContainerDevice{
			Host: cast.ToString(device["PathOnHost"]), Container: cast.ToString(device["PathInContainer"]),
			Permissions: cast.ToString(device["CgroupPermissions"]),
		})
	}
	for _, rawUlimit := range cast.ToSlice(hostConfig["Ulimits"]) {
		ulimit := cast.ToStringMap(rawUlimit)
		detail.Ulimits = append(detail.Ulimits, types.ContainerUlimit{
			Name: cast.ToString(ulimit["Name"]), Soft: cast.ToInt64(ulimit["Soft"]), Hard: cast.ToInt64(ulimit["Hard"]),
		})
	}
	health := cast.ToStringMap(config["Healthcheck"])
	healthTest := r.strings(health["Test"])
	if len(health) > 0 && len(healthTest) > 0 && healthTest[0] != "NONE" {
		detail.Healthcheck = &types.ContainerHealthcheck{
			Test: healthTest, Interval: time.Duration(cast.ToInt64(health["Interval"])),
			Timeout: time.Duration(cast.ToInt64(health["Timeout"])), StartPeriod: time.Duration(cast.ToInt64(health["StartPeriod"])),
			Retries: int(cast.ToInt64(health["Retries"])),
		}
	}
	detail.Network = cast.ToString(hostConfig["NetworkMode"])
	if detail.Network == "default" {
		detail.Network = "bridge"
	}
	networks := cast.ToStringMap(cast.ToStringMap(info["NetworkSettings"])["Networks"])
	networkName := detail.Network
	if _, ok := networks[networkName]; !ok {
		names := make([]string, 0, len(networks))
		for name := range networks {
			names = append(names, name)
		}
		slices.Sort(names)
		if len(names) > 0 {
			networkName = names[0]
		}
	}
	if rawNetwork, ok := networks[networkName]; ok {
		network := cast.ToStringMap(rawNetwork)
		detail.Network = networkName
		if networkName != "bridge" && networkName != "host" && networkName != "none" {
			detail.NetworkAliases = r.strings(network["Aliases"])
			detail.StaticIP = cast.ToString(network["IPAddress"])
		}
	}
	for port, value := range cast.ToStringMap(hostConfig["PortBindings"]) {
		containerPort, protocol, _ := strings.Cut(port, "/")
		if protocol == "" {
			protocol = "tcp"
		}
		bindings, _ := value.([]any)
		for _, binding := range bindings {
			entry := cast.ToStringMap(binding)
			host := netip.IPv4Unspecified()
			if value := cast.ToString(entry["HostIp"]); value != "" && value != "0.0.0.0" {
				host, _ = netip.ParseAddr(value)
			}
			detail.Ports = append(detail.Ports, types.MigrationContainerPort{
				Container: cast.ToUint(containerPort), Host: host, HostPort: cast.ToUint(entry["HostPort"]), Protocol: protocol,
			})
		}
	}
	for _, rawMount := range cast.ToSlice(info["Mounts"]) {
		mount := cast.ToStringMap(rawMount)
		mode := cast.ToString(mount["Mode"])
		if _, exists := mount["RW"]; exists && !cast.ToBool(mount["RW"]) && mode == "" {
			mode = "ro"
		}
		detail.Volumes = append(detail.Volumes, types.MigrationContainerVolume{
			Type: strings.ToLower(cast.ToString(mount["Type"])), Name: cast.ToString(mount["Name"]), Source: cast.ToString(mount["Source"]),
			Destination: cast.ToString(mount["Destination"]), Mode: mode,
		})
	}
	return detail
}

func (r *toolboxMigrationSourceRepo) SetRunning(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	detail *types.MigrationSourceDetail,
	running bool,
) error {
	adapter, err := r.adapter(conn)
	if err != nil {
		return err
	}
	return adapter.SetRunning(ctx, detail, running)
}

func (r *toolboxMigrationSourceRepo) Prepare(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	detail *types.MigrationSourceDetail,
) (*types.MigrationArtifact, error) {
	adapter, err := r.adapter(conn)
	if err != nil {
		return nil, err
	}
	return adapter.Prepare(ctx, detail)
}

func (r *toolboxMigrationSourceRepo) composeImages(content string, envs []types.KV) map[string]string {
	var document map[string]any
	if yaml.Unmarshal([]byte(content), &document) != nil {
		return nil
	}
	services, _ := document["services"].(map[string]any)
	values := types.KVToMap(envs)
	images := make(map[string]string)
	for _, raw := range services {
		service, _ := raw.(map[string]any)
		source, _ := service["image"].(string)
		image := os.Expand(source, func(key string) string {
			name, fallback, hasFallback := strings.Cut(key, ":-")
			if value := values[name]; value != "" {
				return value
			}
			if hasFallback {
				return fallback
			}
			return values[key]
		})
		if source != "" && image != "" {
			images[source] = image
		}
	}
	return images
}

func (r *toolboxMigrationSourceRepo) Download(
	ctx context.Context,
	conn *request.ToolboxMigrationConnection,
	artifact *types.MigrationArtifact,
	target string,
) error {
	adapter, err := r.adapter(conn)
	if err != nil {
		return err
	}
	return adapter.Download(ctx, artifact, target)
}

func (r *toolboxMigrationSourceRepo) download(
	ctx context.Context,
	artifact *types.MigrationArtifact,
	target string,
	downloader func(context.Context, string, string) error,
) error {
	paths := artifact.RemotePaths
	if len(paths) == 0 {
		paths = []string{artifact.RemotePath}
	}
	if len(paths) == 1 {
		return r.downloadWithRetry(ctx, downloader, paths[0], target)
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(target), ".migration-download-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	files := make([]string, 0, len(paths))
	for i, remotePath := range paths {
		localPath := filepath.Join(tmpDir, fmt.Sprintf("%d-%s", i, filepath.Base(remotePath)))
		if err = r.downloadWithRetry(ctx, downloader, remotePath, localPath); err != nil {
			return err
		}
		files = append(files, localPath)
	}
	return r.packBundle(files, target)
}

func (r *toolboxMigrationSourceRepo) downloadWithRetry(
	ctx context.Context,
	downloader func(context.Context, string, string) error,
	remotePath string,
	target string,
) error {
	var lastErr error
	for range 180 {
		lastErr = downloader(ctx, remotePath, target)
		if lastErr == nil {
			info, statErr := os.Stat(target)
			if statErr == nil && info.Size() > 0 {
				if validateErr := r.validateDownload(target); validateErr == nil {
					return nil
				} else {
					lastErr = validateErr
				}
			} else if statErr != nil {
				lastErr = statErr
			} else {
				lastErr = errors.New("downloaded backup is empty")
			}
		}
		_ = os.Remove(target)
		if err := r.wait(ctx, time.Second); err != nil {
			return err
		}
	}
	return fmt.Errorf("download source backup: %w", lastErr)
}

func (r *toolboxMigrationSourceRepo) validateDownload(path string) error {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".gz") {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			_ = file.Close()
			return err
		}
		_, copyErr := io.Copy(io.Discard, reader)
		closeErr := reader.Close()
		_ = file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	if strings.HasSuffix(lower, ".tar") {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		reader := tar.NewReader(file)
		for {
			_, err = reader.Next()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			if _, err = io.Copy(io.Discard, reader); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *toolboxMigrationSourceRepo) packBundle(files []string, target string) error {
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(output)
	tarWriter := tar.NewWriter(gz)
	for _, filePath := range files {
		info, statErr := os.Stat(filePath)
		if statErr != nil {
			_ = tarWriter.Close()
			_ = gz.Close()
			_ = output.Close()
			return statErr
		}
		header, headerErr := tar.FileInfoHeader(info, "")
		if headerErr != nil {
			_ = tarWriter.Close()
			_ = gz.Close()
			_ = output.Close()
			return headerErr
		}
		header.Name = filepath.Base(filePath)
		if headerErr = tarWriter.WriteHeader(header); headerErr != nil {
			_ = tarWriter.Close()
			_ = gz.Close()
			_ = output.Close()
			return headerErr
		}
		input, openErr := os.Open(filePath)
		if openErr != nil {
			_ = tarWriter.Close()
			_ = gz.Close()
			_ = output.Close()
			return openErr
		}
		_, copyErr := io.Copy(tarWriter, input)
		_ = input.Close()
		if copyErr != nil {
			_ = tarWriter.Close()
			_ = gz.Close()
			_ = output.Close()
			return copyErr
		}
	}
	if err = tarWriter.Close(); err != nil {
		_ = gz.Close()
		_ = output.Close()
		return err
	}
	if err = gz.Close(); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func (r *toolboxMigrationSourceRepo) wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *toolboxMigrationSourceRepo) runtimeVersion(version string) uint {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "php"))
	parts := strings.FieldsFunc(version, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	if len(parts) < 2 {
		return 0
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil || major < 0 || minor < 0 {
		return 0
	}
	return uint(major*10 + minor)
}

func (r *toolboxMigrationSourceRepo) projectType(typ string) types.ProjectType {
	switch strings.ToLower(typ) {
	case "php":
		return types.ProjectTypePHP
	case "java", "springboot", "spring_boot":
		return types.ProjectTypeJava
	case "go", "golang":
		return types.ProjectTypeGo
	case "python":
		return types.ProjectTypePython
	case "node", "nodejs":
		return types.ProjectTypeNodejs
	case "net", "dotnet", ".net":
		return types.ProjectTypeDotnet
	default:
		return types.ProjectTypeGeneral
	}
}

func (r *toolboxMigrationSourceRepo) environment(value any) []types.KV {
	if text, ok := value.(string); ok {
		return types.SliceToKV(strings.Split(text, "\n"))
	}
	if values, ok := value.([]any); ok {
		result := make([]types.KV, 0, len(values))
		lines := make([]string, 0, len(values))
		for _, item := range values {
			if row := cast.ToStringMap(item); len(row) > 0 {
				key := lo.CoalesceOrEmpty(cast.ToString(row["key"]), cast.ToString(row["Key"]), cast.ToString(row["k"]))
				if key != "" {
					result = append(result, types.KV{
						Key:   key,
						Value: lo.CoalesceOrEmpty(cast.ToString(row["value"]), cast.ToString(row["Value"]), cast.ToString(row["v"])),
					})
					continue
				}
			}
			lines = append(lines, fmt.Sprint(item))
		}
		return append(result, types.SliceToKV(lines)...)
	}
	return types.MapToKV(cast.ToStringMapString(value))
}

func (r *toolboxMigrationSourceRepo) strings(value any) []string {
	switch data := value.(type) {
	case []string:
		return data
	case []any:
		return lo.Map(data, func(item any, _ int) string { return cast.ToString(item) })
	case string:
		if data == "" {
			return nil
		}
		return []string{data}
	default:
		return nil
	}
}

func (r *toolboxMigrationSourceRepo) parseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "0001-") || value == "0000-00-00" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

func (r *toolboxMigrationSourceRepo) safeFileName(name string) string {
	var result strings.Builder
	for _, char := range name {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.' {
			result.WriteRune(char)
		} else {
			result.WriteRune('_')
		}
	}
	if result.Len() == 0 {
		return "resource"
	}
	return result.String()
}
