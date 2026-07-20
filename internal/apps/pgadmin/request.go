package pgadmin

type UpdatePort struct {
	Port uint `form:"port" json:"port" validate:"required && number && min:1 && max:65535"`
}

type ResetPassword struct {
	Password string `form:"password" json:"password" validate:"required && password"`
}

type UpdateUsername struct {
	Username string `form:"username" json:"username" validate:"required && email"`
}
