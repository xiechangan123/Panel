package types

type Load struct {
	Load1  []float64 `json:"load1"`
	Load5  []float64 `json:"load5"`
	Load15 []float64 `json:"load15"`
}

type CPU struct {
	Percent []string `json:"percent"`
}

type Mem struct {
	Total     string   `json:"total"`
	Available []string `json:"available"`
	Used      []string `json:"used"`
}

type SWAP struct {
	Total string   `json:"total"`
	Used  []string `json:"used"`
	Free  []string `json:"free"`
}

// Network 网卡数据
type Network struct {
	Name string   `json:"name"`
	Sent []string `json:"sent"`
	Recv []string `json:"recv"`
	Tx   []string `json:"tx"`
	Rx   []string `json:"rx"`
}

// DiskIO 磁盘IO数据
type DiskIO struct {
	Name       string   `json:"name"`
	ReadBytes  []string `json:"read_bytes"`
	WriteBytes []string `json:"write_bytes"`
	ReadSpeed  []string `json:"read_speed"`
	WriteSpeed []string `json:"write_speed"`
}

type MonitorDetail struct {
	Times        []string       `json:"times"`
	Load         Load           `json:"load"`
	CPU          CPU            `json:"cpu"`
	Mem          Mem            `json:"mem"`
	SWAP         SWAP           `json:"swap"`
	Net          []Network      `json:"net"`
	DiskIO       []DiskIO       `json:"disk_io"`
	TopProcesses []TopProcesses `json:"top_processes"`
}
