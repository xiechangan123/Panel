package openlitespeed

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// SetPHP 切换 PHP 版本运行协议
type SetPHP struct {
	Version uint `form:"version" json:"version" validate:"required"`
	LSAPI   bool `form:"lsapi" json:"lsapi"`
}

// PHPProtocol PHP 版本运行协议信息
type PHPProtocol struct {
	Version uint `json:"version"`
	LSPHP   bool `json:"lsphp"` // lsphp 二进制是否存在
	LSAPI   bool `json:"lsapi"` // 是否以 LSAPI 协议运行
}
