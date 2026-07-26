package route

import (
	"net/http"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxSSHRoutes 工具箱-SSH 路由
func ToolboxSSHRoutes(toolboxSSHService *service.ToolboxSSHService) Endpoints {
	svc := toolboxSSHService

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_ssh/info", Handler: svc.GetInfo},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/port", Handler: svc.UpdatePort},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/password_auth", Handler: svc.UpdatePasswordAuth},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/pubkey_auth", Handler: svc.UpdatePubKeyAuth},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/root_login", Handler: svc.UpdateRootLogin},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/root_password", Handler: svc.UpdateRootPassword},
		{Method: http.MethodGet, Path: "/api/toolbox_ssh/root_key", Handler: svc.GetRootKey},
		{Method: http.MethodPost, Path: "/api/toolbox_ssh/root_key", Handler: svc.GenerateRootKey},
	}
}
