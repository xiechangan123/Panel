package request

type DatabaseList struct {
	Paginate
	Type string `form:"type" json:"type" query:"type"`
}

type DatabaseCreate struct {
	ServerID   uint   `form:"server_id" json:"server_id" validate:"required && exists:database_servers,id"`
	Name       string `form:"name" json:"name" validate:"required && regex:\"^[A-Za-z0-9_.-]{1,64}$\""`
	CreateUser bool   `form:"create_user" json:"create_user"`
	Username   string `form:"username" json:"username" validate:"required_if:CreateUser,true && not_in:root,admin && regex:\"^[A-Za-z0-9_.-]{1,63}$\""`
	Password   string `form:"password" json:"password" validate:"required_if:CreateUser,true"`
	Host       string `form:"host" json:"host"`
	Comment    string `form:"comment" json:"comment"`
}

type DatabaseDelete struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required && exists:database_servers,id"`
	Name     string `form:"name" json:"name" validate:"required && regex:\"^[A-Za-z0-9_.-]{1,64}$\""`
}

type DatabaseComment struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required && exists:database_servers,id"`
	Name     string `form:"name" json:"name" validate:"required && regex:\"^[A-Za-z0-9_.-]{1,64}$\""`
	Comment  string `form:"comment" json:"comment"`
}
