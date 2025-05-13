package request

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
	TwoFA string `json:"two_fa" validate:"required"`
}

type UserUpdateTwoFA struct {
	ID    uint   `json:"id" validate:"required|exists:users,id"`
	TwoFA string `json:"two_fa" validate:"required"`
	Code  string `json:"code" validate:"required"`
}

type UserDelete struct {
	ID uint `json:"id" validate:"required|exists:users,id"`
}
