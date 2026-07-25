package request

type AlertRuleCreate struct {
	Name      string  `json:"name" form:"name" validate:"required"`
	Type      string  `json:"type" form:"type" validate:"required && in:cpu,memory,swap,load1,load5,load15,disk,disk_inode,disk_read,disk_write,net_in,net_out,website_5xx,website_error,service,project,container,app,database,cert_expire,website_expire"`
	Target    string  `json:"target" form:"target"`
	Operator  string  `json:"operator" form:"operator" validate:"required && in:gt,gte,lt,lte"`
	Threshold float64 `json:"threshold" form:"threshold"`
	Duration  uint    `json:"duration" form:"duration" validate:"min:1 && max:60"`
	Silence   uint    `json:"silence" form:"silence" validate:"min:0 && max:1440"`
	Channels  []uint  `json:"channels" form:"channels"`
	Enabled   bool    `json:"enabled" form:"enabled"`
}

type AlertRuleUpdate struct {
	ID        uint    `json:"id" form:"id" uri:"id" validate:"required && exists:alert_rules,id"`
	Name      string  `json:"name" form:"name" validate:"required"`
	Type      string  `json:"type" form:"type" validate:"required && in:cpu,memory,swap,load1,load5,load15,disk,disk_inode,disk_read,disk_write,net_in,net_out,website_5xx,website_error,service,project,container,app,database,cert_expire,website_expire"`
	Target    string  `json:"target" form:"target"`
	Operator  string  `json:"operator" form:"operator" validate:"required && in:gt,gte,lt,lte"`
	Threshold float64 `json:"threshold" form:"threshold"`
	Duration  uint    `json:"duration" form:"duration" validate:"min:1 && max:60"`
	Silence   uint    `json:"silence" form:"silence" validate:"min:0 && max:1440"`
	Channels  []uint  `json:"channels" form:"channels"`
	Enabled   bool    `json:"enabled" form:"enabled"`
}
