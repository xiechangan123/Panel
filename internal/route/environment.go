package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// EnvironmentRoutes 运行环境路由
func EnvironmentRoutes(i do.Injector) (Endpoints, error) {
	environment := do.MustInvoke[*service.EnvironmentService](i)
	environmentGo := do.MustInvoke[*service.EnvironmentGoService](i)
	environmentJava := do.MustInvoke[*service.EnvironmentJavaService](i)
	environmentNodejs := do.MustInvoke[*service.EnvironmentNodejsService](i)
	environmentPHP := do.MustInvoke[*service.EnvironmentPHPService](i)
	environmentPython := do.MustInvoke[*service.EnvironmentPythonService](i)
	environmentDotnet := do.MustInvoke[*service.EnvironmentDotnetService](i)

	tags := []string{"运行环境"}

	return Endpoints{
		// 顶层
		{Method: http.MethodGet, Path: "/api/environment/types", Handler: environment.Types,
			Summary: "运行环境类型", Tags: tags},
		{Method: http.MethodGet, Path: "/api/environment/list", Handler: environment.List,
			Summary: "运行环境列表", Tags: tags},
		{Method: http.MethodPost, Path: "/api/environment/install", Handler: environment.Install,
			Summary: "安装运行环境", Tags: tags, Request: request.EnvironmentAction{}},
		{Method: http.MethodPost, Path: "/api/environment/uninstall", Handler: environment.Uninstall,
			Summary: "卸载运行环境", Tags: tags, Request: request.EnvironmentAction{}},
		{Method: http.MethodPost, Path: "/api/environment/update", Handler: environment.Update,
			Summary: "更新运行环境", Tags: tags, Request: request.EnvironmentAction{}},
		{Method: http.MethodGet, Path: "/api/environment/is_installed", Handler: environment.IsInstalled,
			Summary: "检查运行环境是否已安装", Tags: tags, Request: request.EnvironmentAction{}},

		// Go
		{Method: http.MethodPost, Path: "/api/environment/go/{slug}/set_cli", Handler: environmentGo.SetCli,
			Summary: "设置 Go 命令行", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodGet, Path: "/api/environment/go/{slug}/proxy", Handler: environmentGo.GetProxy,
			Summary: "获取 Go 代理", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodPost, Path: "/api/environment/go/{slug}/proxy", Handler: environmentGo.SetProxy,
			Summary: "设置 Go 代理", Tags: tags, Request: request.EnvironmentProxy{}},

		// Java
		{Method: http.MethodPost, Path: "/api/environment/java/{slug}/set_cli", Handler: environmentJava.SetCli,
			Summary: "设置 Java 命令行", Tags: tags, Request: request.EnvironmentSlug{}},

		// Node.js
		{Method: http.MethodPost, Path: "/api/environment/nodejs/{slug}/set_cli", Handler: environmentNodejs.SetCli,
			Summary: "设置 Node.js 命令行", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodGet, Path: "/api/environment/nodejs/{slug}/registry", Handler: environmentNodejs.GetRegistry,
			Summary: "获取 Node.js 镜像源", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodPost, Path: "/api/environment/nodejs/{slug}/registry", Handler: environmentNodejs.SetRegistry,
			Summary: "设置 Node.js 镜像源", Tags: tags, Request: request.EnvironmentRegistry{}},

		// PHP
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/set_cli", Handler: environmentPHP.SetCli,
			Summary: "设置 PHP 命令行", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/phpinfo", Handler: environmentPHP.PHPInfo,
			Summary: "获取 phpinfo", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/config", Handler: environmentPHP.GetConfig,
			Summary: "获取 PHP 配置", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/config", Handler: environmentPHP.UpdateConfig,
			Summary: "更新 PHP 配置", Tags: tags, Request: request.EnvironmentPHPUpdateConfig{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/fpm_config", Handler: environmentPHP.GetFPMConfig,
			Summary: "获取 PHP-FPM 配置", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/fpm_config", Handler: environmentPHP.UpdateFPMConfig,
			Summary: "更新 PHP-FPM 配置", Tags: tags, Request: request.EnvironmentPHPUpdateConfig{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/load", Handler: environmentPHP.Load,
			Summary: "获取 PHP-FPM 负载", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/log", Handler: environmentPHP.Log,
			Summary: "获取 PHP 日志路径", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/slow_log", Handler: environmentPHP.SlowLog,
			Summary: "获取 PHP 慢日志路径", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/modules", Handler: environmentPHP.ModuleList,
			Summary: "PHP 扩展列表", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/modules", Handler: environmentPHP.InstallModule,
			Summary: "安装 PHP 扩展", Tags: tags, Request: request.EnvironmentPHPModule{}},
		{Method: http.MethodDelete, Path: "/api/environment/php/{version}/modules", Handler: environmentPHP.UninstallModule,
			Summary: "卸载 PHP 扩展", Tags: tags, Request: request.EnvironmentPHPModule{}},
		{Method: http.MethodGet, Path: "/api/environment/php/{version}/config_tune", Handler: environmentPHP.GetConfigTune,
			Summary: "获取 PHP 配置调优", Tags: tags, Request: request.EnvironmentPHPVersion{}},
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/config_tune", Handler: environmentPHP.UpdateConfigTune,
			Summary: "更新 PHP 配置调优", Tags: tags, Request: request.EnvironmentPHPConfigTune{}},
		{Method: http.MethodPost, Path: "/api/environment/php/{version}/clean_session", Handler: environmentPHP.CleanSession,
			Summary: "清理 PHP Session", Tags: tags, Request: request.EnvironmentPHPVersion{}},

		// Python
		{Method: http.MethodPost, Path: "/api/environment/python/{slug}/set_cli", Handler: environmentPython.SetCli,
			Summary: "设置 Python 命令行", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodGet, Path: "/api/environment/python/{slug}/mirror", Handler: environmentPython.GetMirror,
			Summary: "获取 Python 镜像源", Tags: tags, Request: request.EnvironmentSlug{}},
		{Method: http.MethodPost, Path: "/api/environment/python/{slug}/mirror", Handler: environmentPython.SetMirror,
			Summary: "设置 Python 镜像源", Tags: tags, Request: request.EnvironmentMirror{}},

		// .NET
		{Method: http.MethodPost, Path: "/api/environment/dotnet/{slug}/set_cli", Handler: environmentDotnet.SetCli,
			Summary: "设置 .NET 命令行", Tags: tags, Request: request.EnvironmentSlug{}},
	}, nil
}
