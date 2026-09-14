package data

import (
	"context"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/firewall"
	"github.com/acepanel/panel/v3/pkg/os"
)

type safeRepo struct {
	ssh string
}

func NewSafeRepo() biz.SafeRepo {
	var ssh string
	if os.IsRHEL() {
		ssh = "sshd"
	} else {
		ssh = "ssh"
	}
	return &safeRepo{
		ssh: ssh,
	}
}

func (r *safeRepo) GetPingStatus(ctx context.Context) (bool, error) {
	fw := firewall.NewFirewall(ctx)
	return fw.PingStatus(ctx)
}

func (r *safeRepo) FirewallRunning(ctx context.Context) (bool, error) {
	fw := firewall.NewFirewall(ctx)
	return fw.Status(ctx)
}

func (r *safeRepo) SetPingStatus(ctx context.Context, status bool) error {
	fw := firewall.NewFirewall(ctx)
	return fw.UpdatePingStatus(ctx, status)
}
