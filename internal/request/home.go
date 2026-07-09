package request

type HomeCurrent struct {
	Nets  []string `json:"nets" form:"nets"`
	Disks []string `json:"disks" form:"disks"`
}

type HomeTopProcesses struct {
	Type string `json:"type" form:"type" query:"type" validate:"in:cpu,memory,disk_io"`
}
