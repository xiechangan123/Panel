package request

type UserID struct {
	ID uint `json:"id" validate:"required|exists:users,id"`
}

type UserLogin struct {
	Username  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
	SafeLogin bool   `json:"safe_login"`
	PassCode  string `json:"pass_code"`
}

type UserIsTwoFA struct {
	Username string `uri:"username" validate:"required"`
}

type UserCreate struct {
	Username string `json:"username" validate:"required|notExists:users,username"`
	Password string `json:"password" validate:"required|password"`
}

type UserUpdatePassword struct {
	ID       uint   `json:"id" validate:"required|exists:users,id"`
	Password string `json:"password" validate:"required|password"`
}

type UserUpdateEmail struct {
	ID    uint   `json:"id" validate:"required|exists:users,id"`
	Email string `json:"email" validate:"required|email"`
}

type UserUpdateTwoFA struct {
	ID     uint   `uri:"id" validate:"required|exists:users,id"`
	Secret string `json:"secret"`
	Code   string `json:"code"`
}
