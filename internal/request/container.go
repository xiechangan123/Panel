package request

import "github.com/acepanel/panel/v3/pkg/types"

type ContainerID struct {
	ID string `json:"id" form:"id" validate:"required"`
}

type ContainerRename struct {
	ID   string `form:"id" json:"id" validate:"required"`
	Name string `form:"name" json:"name" validate:"required && regex:\"^([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9._-]{0,126}[A-Za-z0-9_-])$\""`
}

type ContainerCreate struct {
	Name            string                           `form:"name" json:"name"`
	Image           string                           `form:"image" json:"image" validate:"required"`
	Hostname        string                           `form:"hostname" json:"hostname"`
	WorkingDir      string                           `form:"working_dir" json:"working_dir"`
	User            string                           `form:"user" json:"user"`
	Background      bool                             `form:"background" json:"background"`
	Ports           []types.ContainerPort            `form:"ports" json:"ports"`
	Network         string                           `form:"network" json:"network"`
	NetworkAliases  []string                         `form:"network_aliases" json:"network_aliases"`
	StaticIP        string                           `form:"static_ip" json:"static_ip"`
	Volumes         []types.ContainerContainerVolume `form:"volumes" json:"volumes"`
	Labels          []types.KV                       `form:"labels" json:"labels"`
	Env             []types.KV                       `form:"env" json:"env"`
	Entrypoint      []string                         `form:"entrypoint" json:"entrypoint"`
	Command         []string                         `form:"command" json:"command"`
	RestartPolicy   string                           `form:"restart_policy" json:"restart_policy" validate:"in:no,always,on-failure,unless-stopped"`
	AutoRemove      bool                             `form:"auto_remove" json:"auto_remove"`
	Privileged      bool                             `form:"privileged" json:"privileged"`
	ReadonlyRootfs  bool                             `form:"readonly_rootfs" json:"readonly_rootfs"`
	DNS             []string                         `form:"dns" json:"dns"`
	ExtraHosts      []string                         `form:"extra_hosts" json:"extra_hosts"`
	CapAdd          []string                         `form:"cap_add" json:"cap_add"`
	CapDrop         []string                         `form:"cap_drop" json:"cap_drop"`
	Devices         []types.ContainerDevice          `form:"devices" json:"devices"`
	SecurityOpt     []string                         `form:"security_opt" json:"security_opt"`
	Sysctls         []types.KV                       `form:"sysctls" json:"sysctls"`
	Ulimits         []types.ContainerUlimit          `form:"ulimits" json:"ulimits"`
	Tmpfs           []types.KV                       `form:"tmpfs" json:"tmpfs"`
	ShmSize         int64                            `form:"shm_size" json:"shm_size"`
	Init            bool                             `form:"init" json:"init"`
	StopSignal      string                           `form:"stop_signal" json:"stop_signal"`
	StopTimeout     int                              `form:"stop_timeout" json:"stop_timeout"`
	Healthcheck     *types.ContainerHealthcheck      `form:"healthcheck" json:"healthcheck"`
	OpenStdin       bool                             `form:"openStdin" json:"open_stdin"`
	PublishAllPorts bool                             `form:"publish_all_ports" json:"publish_all_ports"`
	Tty             bool                             `form:"tty" json:"tty"`
	CPUShares       int64                            `form:"cpu_shares" json:"cpu_shares"`
	CPUs            float64                          `form:"cpus" json:"cpus"`
	Memory          int64                            `form:"memory" json:"memory"`
}
