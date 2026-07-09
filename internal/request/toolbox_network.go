package request

// ToolboxNetworkList 网络连接列表请求
type ToolboxNetworkList struct {
	Page    uint   `json:"page" form:"page" query:"page"`
	Limit   uint   `json:"limit" form:"limit" query:"limit"`
	Sort    string `json:"sort" form:"sort" query:"sort" validate:"in:type,pid,process"`
	Order   string `json:"order" form:"order" query:"order" validate:"in:asc,desc"`
	State   string `json:"state" form:"state" query:"state"`       // 逗号分隔多选: LISTEN,ESTABLISHED
	PID     string `json:"pid" form:"pid" query:"pid"`             // PID 搜索
	Process string `json:"process" form:"process" query:"process"` // 进程名称搜索
	Port    string `json:"port" form:"port" query:"port"`          // 端口搜索
}
