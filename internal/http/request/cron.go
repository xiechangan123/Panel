package request

type CronCreate struct {
	Name    string   `form:"name" json:"name" validate:"required|notExists:crons,name"`
	Type    string   `form:"type" json:"type" validate:"required"`
	Time    string   `form:"time" json:"time" validate:"required|cron"`
	Script  string   `form:"script" json:"script"`
	SubType string   `form:"sub_type" json:"sub_type" validate:"requiredIf:Type,backup,cutoff"`
	Storage uint     `form:"storage" json:"storage"`
	Targets []string `form:"targets" json:"targets" validate:"requiredIf:Type,backup,cutoff"`
	Keep    uint     `form:"keep" json:"keep" validate:"required"`
}

type CronUpdate struct {
	ID      uint     `form:"id" json:"id" validate:"required|exists:crons,id"`
	Name    string   `form:"name" json:"name" validate:"required"`
	Type    string   `form:"type" json:"type" validate:"required"`
	Time    string   `form:"time" json:"time" validate:"required|cron"`
	Script  string   `form:"script" json:"script"`
	SubType string   `form:"sub_type" json:"sub_type"`
	Storage uint     `form:"storage" json:"storage"`
	Targets []string `form:"targets" json:"targets"`
	Keep    uint     `form:"keep" json:"keep"`
}

type CronStatus struct {
	ID     uint `form:"id" json:"id" validate:"required|exists:crons,id"`
	Status bool `form:"status" json:"status"`
}
