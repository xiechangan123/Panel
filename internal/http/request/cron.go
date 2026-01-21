package request

type CronCreate struct {
	Name          string `form:"name" json:"name" validate:"required|notExists:crons,name"`
	Type          string `form:"type" json:"type" validate:"required"`
	Time          string `form:"time" json:"time" validate:"required|cron"`
	Script        string `form:"script" json:"script"`
	BackupType    string `form:"backup_type" json:"backup_type" validate:"requiredIf:Type,backup"`
	BackupStorage uint   `form:"backup_storage" json:"backup_storage"`
	Target        string `form:"target" json:"target" validate:"requiredIf:Type,backup,cutoff"`
	Keep          uint   `form:"keep" json:"keep" validate:"required"`
}

type CronUpdate struct {
	ID     uint   `form:"id" json:"id" validate:"required|exists:crons,id"`
	Name   string `form:"name" json:"name" validate:"required"`
	Time   string `form:"time" json:"time" validate:"required|cron"`
	Script string `form:"script" json:"script" validate:"required"`
}

type CronStatus struct {
	ID     uint `form:"id" json:"id" validate:"required|exists:crons,id"`
	Status bool `form:"status" json:"status"`
}
