package request

type TaskIDs struct {
	IDs []uint `json:"ids" form:"ids" validate:"required && unique"`
}
