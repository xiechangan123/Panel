package rocketmq

// UpdateConfig RocketMQ 配置更新
type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// ConfigTune RocketMQ 配置调整
type ConfigTune struct {
	// Broker 基础
	BrokerName    string `form:"broker_name" json:"broker_name"`         // brokerName
	ListenPort    string `form:"listen_port" json:"listen_port"`         // listenPort
	NamesrvAddr   string `form:"namesrv_addr" json:"namesrv_addr"`       // namesrvAddr
	BrokerRole    string `form:"broker_role" json:"broker_role"`         // brokerRole
	FlushDiskType string `form:"flush_disk_type" json:"flush_disk_type"` // flushDiskType
	// 存储
	StorePathRootDir   string `form:"store_path_root_dir" json:"store_path_root_dir"`     // storePathRootDir
	StorePathCommitLog string `form:"store_path_commit_log" json:"store_path_commit_log"` // storePathCommitLog
	MaxMessageSize     string `form:"max_message_size" json:"max_message_size"`           // maxMessageSize
	// JVM - NameServer
	NamesrvHeapInitSize string `form:"namesrv_heap_init_size" json:"namesrv_heap_init_size"` // -Xms (namesrv)
	NamesrvHeapMaxSize  string `form:"namesrv_heap_max_size" json:"namesrv_heap_max_size"`   // -Xmx (namesrv)
	// JVM - Broker
	BrokerHeapInitSize string `form:"broker_heap_init_size" json:"broker_heap_init_size"` // -Xms (broker)
	BrokerHeapMaxSize  string `form:"broker_heap_max_size" json:"broker_heap_max_size"`   // -Xmx (broker)
}
