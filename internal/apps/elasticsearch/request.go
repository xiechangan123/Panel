package elasticsearch

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// ConfigTune ElasticSearch 配置调整
type ConfigTune struct {
	// 集群
	ClusterName   string `form:"cluster_name" json:"cluster_name"`
	NodeName      string `form:"node_name" json:"node_name"`
	NetworkHost   string `form:"network_host" json:"network_host"`
	HTTPPort      string `form:"http_port" json:"http_port"`
	DiscoveryType string `form:"discovery_type" json:"discovery_type"`
	// 路径
	PathData string `form:"path_data" json:"path_data"`
	PathLogs string `form:"path_logs" json:"path_logs"`
	// JVM
	HeapInitSize string `form:"heap_init_size" json:"heap_init_size"`
	HeapMaxSize  string `form:"heap_max_size" json:"heap_max_size"`
}
