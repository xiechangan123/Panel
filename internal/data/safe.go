package data

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/firewall"
	"github.com/acepanel/panel/pkg/os"
)

type safeRepo struct {
	ssh string
	log *slog.Logger
}

func NewSafeRepo(log *slog.Logger) biz.SafeRepo {
	var ssh string
	if os.IsRHEL() {
		ssh = "sshd"
	} else {
		ssh = "ssh"
	}
	return &safeRepo{
		ssh: ssh,
		log: log,
	}
}

func (r *safeRepo) GetPingStatus() (bool, error) {
	fw := firewall.NewFirewall()
	return fw.PingStatus()
}

func (r *safeRepo) UpdatePingStatus(ctx context.Context, status bool) error {
	fw := firewall.NewFirewall()
	running, err := fw.Status()
	if err != nil {
		return err
	}
	if !running {
		return fmt.Errorf("failed to update ping status: firewall is not running")
	}

	if err = fw.UpdatePingStatus(status); err != nil {
		return err
	}

	// 记录日志
	r.log.Info("ping status updated", slog.String("type", biz.OperationTypeSafe), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Bool("status", status))

	return nil
}
