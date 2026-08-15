package rsync

type ModuleName struct {
	Name string `form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
}
