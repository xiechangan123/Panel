// Package common 承载各应用共用的 HTTP 处理逻辑
package common

import (
	"net/http"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

// ServeConfig 返回配置文件原始内容
func ServeConfig(w http.ResponseWriter, path string) {
	config, _ := io.Read(path)

	service.Success(w, config)
}

// SaveConfig 写入配置文件并重启对应服务
func SaveConfig(w http.ResponseWriter, r *http.Request, path, unit string) {
	req, err := service.Bind[request.AppUpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(path, req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart(unit); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}
