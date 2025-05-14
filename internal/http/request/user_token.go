package request

import "net/http"

type UserTokenList struct {
	UserID uint `query:"user_id"`
	Paginate
}

type UserTokenCreate struct {
	UserID    uint     `json:"user_id" validate:"required|exists:users,id"`
	IPs       []string `json:"ips"`
	ExpiredAt int64    `json:"expired_at" validate:"required"`
}

func (r *UserTokenCreate) Rules(_ *http.Request) map[string]string {
	return map[string]string{
		"IPs.*": "required|ip",
	}
}

type UserTokenUpdate struct {
	ID        uint     `uri:"id"`
	IPs       []string `json:"ips"`
	ExpiredAt int64    `json:"expired_at" validate:"required"`
}

func (r *UserTokenUpdate) Rules(_ *http.Request) map[string]string {
	return map[string]string{
		"IPs.*": "required|ip",
	}
}
