package types

import "time"

type BackupAccountInfo struct {
	// S3
	AccessKey string `json:"access_key"` // 访问密钥
	SecretKey string `json:"secret_key"` // 私钥
	Style     string `json:"style"`      // virtual_hosted, path
	Region    string `json:"region"`     // 地区
	Endpoint  string `json:"endpoint"`   // 端点
	Bucket    string `json:"bucket"`     // 存储桶

	// SFTP / WebDAV
	Host     string `json:"host"`     // 主机
	Port     int    `json:"port"`     // 端口
	User     string `json:"user"`     // 用户名
	Password string `json:"password"` // 密码

	Path string `json:"path"` // 路径
}

type BackupFile struct {
	Name string    `json:"name"`
	Path string    `json:"path"`
	Size string    `json:"size"`
	Time time.Time `json:"time"`
}
