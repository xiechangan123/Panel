package biz

import "context"

type SafeRepo interface {
	GetPingStatus() (bool, error)
	UpdatePingStatus(ctx context.Context, status bool) error
}
