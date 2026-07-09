package request

type UserTokenList struct {
	UserID uint `query:"user_id"`
	Paginate
}

type UserTokenCreate struct {
	UserID    uint     `json:"user_id" validate:"required && exists:users,id"`
	IPs       []string `json:"ips" validate:"unique && dive && ipcidr"`
	ExpiredAt int64    `json:"expired_at" validate:"required"`
}

type UserTokenUpdate struct {
	ID        uint     `uri:"id"`
	IPs       []string `json:"ips" validate:"unique && dive && ipcidr"`
	ExpiredAt int64    `json:"expired_at" validate:"required"`
}
