package types

import "time"

type BackupStorageInfo struct {
	// S3
	AccessKey string `json:"access_key"`                             // 访问密钥
	SecretKey string `json:"secret_key"`                             // 私钥
	Style     string `json:"style" validate:"requiredIf:Type,s3"`    // virtual-hosted, path
	Region    string `json:"region"`                                 // 地区
	Endpoint  string `json:"endpoint" validate:"requiredIf:Type,s3"` // 端点
	Scheme    string `json:"scheme" validate:"requiredIf:Type,s3"`   // http, https
	Bucket    string `json:"bucket" validate:"requiredIf:Type,s3"`   // 存储桶

	// SFTP / WebDAV
	URL        string `json:"url"`         // 网址
	Host       string `json:"host"`        // 主机
	Port       int    `json:"port"`        // 端口
	Username   string `json:"username"`    // 用户名
	Password   string `json:"password"`    // 密码
	PrivateKey string `json:"private_key"` // 私钥

	Path string `json:"path"` // 路径
}

type BackupFile struct {
	Name string    `json:"name"`
	Path string    `json:"path"`
	Size string    `json:"size"`
	Time time.Time `json:"time"`
}
