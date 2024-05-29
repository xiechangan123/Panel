package models

import (
	"github.com/goravel/framework/support/carbon"

	"github.com/TheTNB/panel/pkg/acme"
)

type CertDNS struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Name      string          `gorm:"not null" json:"name"` // 备注名称
	Type      string          `gorm:"not null" json:"type"` // DNS 提供商 (dnspod, aliyun, cloudflare)
	Data      acme.DNSParam   `gorm:"type:json;serializer:json" json:"dns_param"`
	CreatedAt carbon.DateTime `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt carbon.DateTime `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`

	Certs []*Cert `gorm:"foreignKey:DNSID" json:"-"`
}

func (CertDNS) TableName() string {
	return "cert_dns"
}
