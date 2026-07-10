package data

import (
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/firewall"
	"github.com/acepanel/panel/v3/pkg/os"
)

type safeRepo struct {
	ssh string
}

func NewSafeRepo(i do.Injector) (biz.SafeRepo, error) {
	var ssh string
	if os.IsRHEL() {
		ssh = "sshd"
	} else {
		ssh = "ssh"
	}
	return &safeRepo{
		ssh: ssh,
	}, nil
}

func (r *safeRepo) GetPingStatus() (bool, error) {
	fw := firewall.NewFirewall()
	return fw.PingStatus()
}

func (r *safeRepo) FirewallRunning() (bool, error) {
	fw := firewall.NewFirewall()
	return fw.Status()
}

func (r *safeRepo) SetPingStatus(status bool) error {
	fw := firewall.NewFirewall()
	return fw.UpdatePingStatus(status)
}
