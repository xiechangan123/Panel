package biz

import (
	"context"
	"errors"
	"log/slog"
)

type SafeRepo interface {
	GetPingStatus(ctx context.Context) (bool, error)
	FirewallRunning(ctx context.Context) (bool, error)
	SetPingStatus(ctx context.Context, status bool) error
}

type SafeUsecase struct {
	repo SafeRepo
	log  *slog.Logger
}

func NewSafeUsecase(repo SafeRepo, log *slog.Logger) *SafeUsecase {
	return &SafeUsecase{repo: repo, log: log}
}

func (uc *SafeUsecase) GetPingStatus(ctx context.Context) (bool, error) {
	return uc.repo.GetPingStatus(ctx)
}

func (uc *SafeUsecase) UpdatePingStatus(ctx context.Context, status bool) error {
	running, err := uc.repo.FirewallRunning(ctx)
	if err != nil {
		return err
	}
	if !running {
		return errors.New("failed to update ping status: firewall is not running")
	}

	if err = uc.repo.SetPingStatus(ctx, status); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("ping status updated", slog.String("type", OperationTypeSafe), slog.Uint64("operator_id", operatorID(ctx)), slog.Bool("status", status))

	return nil
}
