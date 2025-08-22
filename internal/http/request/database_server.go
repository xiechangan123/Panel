package request

type DatabaseServerCreate struct {
	Name     string `form:"name" json:"name" validate:"required|notExists:database_servers,name|regex:^[a-zA-Z0-9_-]+$"`
	Type     string `form:"type" json:"type" validate:"required|in:mysql,postgresql,redis"`
	Host     string `form:"host" json:"host" validate:"required"`
	Port     uint   `form:"port" json:"port" validate:"required|min:1|max:65535"`
	Username string `form:"username" json:"username"`
	Password string `form:"password" json:"password"`
	Remark   string `form:"remark" json:"remark"`
}

type DatabaseServerUpdate struct {
	ID       uint   `form:"id" json:"id" validate:"required|exists:database_servers,id"`
	Name     string `form:"name" json:"name" validate:"required|regex:^[a-zA-Z0-9_-]+$"`
	Host     string `form:"host" json:"host" validate:"required"`
	Port     uint   `form:"port" json:"port" validate:"required|min:1|max:65535"`
	Username string `form:"username" json:"username"`
	Password string `form:"password" json:"password"`
	Remark   string `form:"remark" json:"remark"`
}

type DatabaseServerUpdateRemark struct {
	ID     uint   `form:"id" json:"id" validate:"required|exists:database_servers,id"`
	Remark string `form:"remark" json:"remark"`
}
