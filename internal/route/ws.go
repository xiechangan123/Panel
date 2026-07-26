package route

import (
	"net/http"

	"github.com/acepanel/panel/v3/internal/service"
)

// WsRoutes WebSocket 路由
func WsRoutes(toolboxMigrationService *service.ToolboxMigrationService, wsService *service.WsService) Endpoints {
	ws := wsService
	toolboxMigration := toolboxMigrationService

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/ws/exec", Handler: ws.Exec},
		{Method: http.MethodGet, Path: "/api/ws/pty", Handler: ws.PTY},
		{Method: http.MethodGet, Path: "/api/ws/follow", Handler: ws.Follow},
		{Method: http.MethodGet, Path: "/api/ws/ssh", Handler: ws.Session},
		{Method: http.MethodGet, Path: "/api/ws/ssh/transfer", Handler: ws.SSHTransfer},
		{Method: http.MethodGet, Path: "/api/ws/container/{id}", Handler: ws.ContainerTerminal},
		{Method: http.MethodGet, Path: "/api/ws/container/image/pull", Handler: ws.ContainerImagePull},
		{Method: http.MethodGet, Path: "/api/ws/migration/progress", Handler: toolboxMigration.Progress},
		{Method: http.MethodGet, Path: "/api/ws/cert/obtain", Handler: ws.CertObtain},
		{Method: http.MethodGet, Path: "/api/ws/cert/renew", Handler: ws.CertRenew},
		{Method: http.MethodGet, Path: "/api/ws/panel/update", Handler: ws.PanelUpdate},
	}
}
