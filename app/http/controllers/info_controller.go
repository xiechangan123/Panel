package controllers

import (
	"fmt"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"panel/app/models"
	"panel/packages/helpers"
)

type MenuItem struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Jump  string `json:"jump"`
}

type InfoController struct {
	//Dependent services
}

func NewInfoController() *InfoController {
	return &InfoController{
		//Inject services
	}
}

func (r *InfoController) Name(ctx http.Context) {
	var setting models.Setting
	err := facades.Orm().Query().Where("key", "name").First(&setting)
	if err != nil {
		facades.Log().Error("[面板][InfoController] 查询面板名称失败 ", err)
		Error(ctx, http.StatusInternalServerError, "系统内部错误")
		return
	}

	Success(ctx, http.Json{
		"name": setting.Value,
	})
}

func (r *InfoController) Menu(ctx http.Context) {
	Success(ctx, []MenuItem{
		{Name: "home", Title: "主页", Icon: "layui-icon-home", Jump: "/"},
		{Name: "website", Title: "网站管理", Icon: "layui-icon-website", Jump: "website/list"},
		{Name: "monitor", Title: "资源监控", Icon: "layui-icon-chart-screen", Jump: "monitor"},
		{Name: "safe", Title: "系统安全", Icon: "layui-icon-auz", Jump: "safe"},
		{Name: "file", Title: "文件管理", Icon: "layui-icon-file", Jump: "file"},
		{Name: "cron", Title: "计划任务", Icon: "layui-icon-date", Jump: "cron"},
		{Name: "plugin", Title: "插件中心", Icon: "layui-icon-app", Jump: "plugin"},
		{Name: "setting", Title: "面板设置", Icon: "layui-icon-set", Jump: "setting"},
	})
}

func (r *InfoController) HomePlugins(ctx http.Context) {
	var plugins []models.Plugin
	err := facades.Orm().Query().Where("show", 1).Find(&plugins)
	if err != nil {
		facades.Log().Error("[面板][InfoController] 查询首页插件失败 ", err)
		Error(ctx, http.StatusInternalServerError, "系统内部错误")
		return
	}

	Success(ctx, plugins)
}

func (r *InfoController) NowMonitor(ctx http.Context) {
	Success(ctx, helpers.GetMonitoringInfo())
}

func (r *InfoController) SystemInfo(ctx http.Context) {
	monitorInfo := helpers.GetMonitoringInfo()

	Success(ctx, http.Json{
		"os_name":       monitorInfo.Host.Platform + " " + monitorInfo.Host.PlatformVersion,
		"uptime":        fmt.Sprintf("%.2f", float64(monitorInfo.Host.Uptime)/86400),
		"panel_version": facades.Config().GetString("panel.version"),
	})
}