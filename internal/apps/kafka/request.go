package kafka

// UpdateConfig Kafka 配置更新
type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// ConfigTune Kafka 配置调整
type ConfigTune struct {
	// Broker
	NodeID          string `form:"node_id" json:"node_id"`                     // node.id
	Listeners       string `form:"listeners" json:"listeners"`                 // listeners
	LogDirs         string `form:"log_dirs" json:"log_dirs"`                   // log.dirs
	NumPartitions   string `form:"num_partitions" json:"num_partitions"`       // num.partitions
	RetentionHours  string `form:"retention_hours" json:"retention_hours"`     // log.retention.hours
	LogSegmentBytes string `form:"log_segment_bytes" json:"log_segment_bytes"` // log.segment.bytes
	// JVM
	HeapInitSize string `form:"heap_init_size" json:"heap_init_size"` // -Xms
	HeapMaxSize  string `form:"heap_max_size" json:"heap_max_size"`   // -Xmx
}
