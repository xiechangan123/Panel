package request

type ID struct {
	ID uint `json:"id" form:"id" query:"id" uri:"id" validate:"required|min:1"`
}
