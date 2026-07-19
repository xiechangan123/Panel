package request

type SSHCreate struct {
	Name       string `json:"name" form:"name"`
	Host       string `json:"host" form:"host" validate:"required"`
	Port       uint   `json:"port" form:"port" validate:"required && min:1 && max:65535"`
	AuthMethod string `json:"auth_method" form:"auth_method" validate:"required && in:password,publickey"`
	User       string `json:"user" form:"user" validate:"required"`
	Password   string `json:"password" form:"password" validate:"required_if:AuthMethod,password"`
	Key        string `json:"key" form:"key" validate:"required_if:AuthMethod,publickey"`
	Passphrase string `json:"passphrase" form:"passphrase"`
	Remark     string `json:"remark" form:"remark"`
}

type SSHUpdate struct {
	ID         uint   `form:"id" json:"id" validate:"required && exists:sshes,id"`
	Name       string `json:"name" form:"name"`
	Host       string `json:"host" form:"host" validate:"required"`
	Port       uint   `json:"port" form:"port" validate:"required && min:1 && max:65535"`
	AuthMethod string `json:"auth_method" form:"auth_method" validate:"required && in:password,publickey"`
	User       string `json:"user" form:"user" validate:"required"`
	Password   string `json:"password" form:"password" validate:"required_if:AuthMethod,password"`
	Key        string `json:"key" form:"key" validate:"required_if:AuthMethod,publickey"`
	Passphrase string `json:"passphrase" form:"passphrase"`
	Remark     string `json:"remark" form:"remark"`
}

// SSHFile 文件浏览请求,ID 为 0 表示面板本机
type SSHFile struct {
	ID   uint   `json:"id" form:"id" uri:"id"`
	Path string `json:"path" form:"path" query:"path" validate:"required"`
}

// SSHTransfer 文件传输请求,ID 为 0 表示面板本机
type SSHTransfer struct {
	SrcID   uint   `json:"src_id"`
	SrcPath string `json:"src_path"`
	DstID   uint   `json:"dst_id"`
	DstPath string `json:"dst_path"`
}
