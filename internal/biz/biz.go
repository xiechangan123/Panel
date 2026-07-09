package biz

import (
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/registry"
)

var Package = do.Package(
	registry.Lazy(NewAppUsecase), registry.Lazy(NewBackupUsecase), registry.Lazy(NewBackupAccountUsecase),
	registry.Lazy(NewCacheUsecase), registry.Lazy(NewCertUsecase), registry.Lazy(NewCertAccountUsecase),
	registry.Lazy(NewCertDNSUsecase), registry.Lazy(NewContainerUsecase), registry.Lazy(NewContainerComposeUsecase),
	registry.Lazy(NewContainerImageUsecase), registry.Lazy(NewContainerNetworkUsecase), registry.Lazy(NewContainerVolumeUsecase),
	registry.Lazy(NewCronUsecase), registry.Lazy(NewDatabaseUsecase), registry.Lazy(NewDatabaseRedisUsecase),
	registry.Lazy(NewDatabaseElasticsearchUsecase), registry.Lazy(NewDatabaseServerUsecase), registry.Lazy(NewDatabaseUserUsecase),
	registry.Lazy(NewEnvironmentUsecase), registry.Lazy(NewLogUsecase), registry.Lazy(NewMonitorUsecase),
	registry.Lazy(NewProjectUsecase), registry.Lazy(NewSafeUsecase), registry.Lazy(NewScanEventUsecase),
	registry.Lazy(NewSettingUsecase), registry.Lazy(NewSSHUsecase), registry.Lazy(NewTaskUsecase),
	registry.Lazy(NewTemplateUsecase), registry.Lazy(NewUserUsecase), registry.Lazy(NewUserPasskeyUsecase),
	registry.Lazy(NewUserTokenUsecase), registry.Lazy(NewWebHookUsecase), registry.Lazy(NewWebsiteUsecase),
	registry.Lazy(NewWebsiteStatUsecase),
)
