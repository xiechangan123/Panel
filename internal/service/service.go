package service

import (
	"github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy(NewAppService), do.Lazy(NewBackupService), do.Lazy(NewBackupStorageService),
	do.Lazy(NewCertService), do.Lazy(NewCertAccountService), do.Lazy(NewCertDNSService),
	do.Lazy(NewCliService), do.Lazy(NewContainerService), do.Lazy(NewContainerComposeService),
	do.Lazy(NewContainerImageService), do.Lazy(NewContainerNetworkService), do.Lazy(NewContainerVolumeService),
	do.Lazy(NewCronService), do.Lazy(NewDatabaseService), do.Lazy(NewDatabaseRedisService),
	do.Lazy(NewDatabaseElasticsearchService), do.Lazy(NewDatabaseServerService), do.Lazy(NewDatabaseUserService),
	do.Lazy(NewEnvironmentService), do.Lazy(NewEnvironmentGoService), do.Lazy(NewEnvironmentJavaService),
	do.Lazy(NewEnvironmentNodejsService), do.Lazy(NewEnvironmentPHPService), do.Lazy(NewEnvironmentPythonService),
	do.Lazy(NewEnvironmentDotnetService), do.Lazy(NewFileService), do.Lazy(NewFirewallService),
	do.Lazy(NewFirewallScanService), do.Lazy(NewHomeService), do.Lazy(NewLogService),
	do.Lazy(NewMonitorService), do.Lazy(NewProcessService), do.Lazy(NewProjectService),
	do.Lazy(NewSafeService), do.Lazy(NewSettingService), do.Lazy(NewSSHService),
	do.Lazy(NewSystemctlService), do.Lazy(NewTaskService), do.Lazy(NewTemplateService),
	do.Lazy(NewUserService), do.Lazy(NewUserPasskeyService), do.Lazy(NewUserTokenService),
	do.Lazy(NewWebHookService), do.Lazy(NewWebsiteService), do.Lazy(NewWebsiteStatService),
	do.Lazy(NewToolboxNetworkService), do.Lazy(NewToolboxSystemService), do.Lazy(NewToolboxBenchmarkService),
	do.Lazy(NewToolboxSSHService), do.Lazy(NewToolboxDiskService), do.Lazy(NewToolboxLogService),
	do.Lazy(NewToolboxMigrationService), do.Lazy(NewWsService),
)
