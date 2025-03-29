package request

import "github.com/tnb-labs/panel/pkg/acme"

type CertDNSCreate struct {
	Type acme.DnsType  `form:"type" json:"type" validate:"required|in:aliyun,tencent,huawei,cloudflare,godaddy,gcore,porkbun,namecheap,namesilo,namecom"`
	Name string        `form:"name" json:"name" validate:"required"`
	Data acme.DNSParam `form:"data" json:"data" validate:"required"`
}

type CertDNSUpdate struct {
	ID   uint          `form:"id" json:"id" validate:"required|exists:cert_dns,id"`
	Type acme.DnsType  `form:"type" json:"type" validate:"required|in:aliyun,tencent,huawei,cloudflare,godaddy,gcore,porkbun,namecheap,namesilo,namecom"`
	Name string        `form:"name" json:"name" validate:"required"`
	Data acme.DNSParam `form:"data" json:"data" validate:"required"`
}
