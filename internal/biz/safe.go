package biz

import "context"

type SafeRepo interface {
	GetPingStatus() (bool, error)
	UpdatePingStatus(ctx context.Context, status bool) error
}

type SafeUsecase struct {
	repo SafeRepo
}

func NewSafeUsecase(repo SafeRepo) *SafeUsecase {
	return &SafeUsecase{repo: repo}
}

func (uc *SafeUsecase) GetPingStatus() (bool, error) {
	return uc.repo.GetPingStatus()
}

func (uc *SafeUsecase) UpdatePingStatus(ctx context.Context, status bool) error {
	return uc.repo.UpdatePingStatus(ctx, status)
}
