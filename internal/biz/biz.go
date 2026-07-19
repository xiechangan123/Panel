package biz

import (
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/registry"
)

var Package = do.Package(
	do.Lazy(NewAppUsecase), registry.Lazy2(NewBackupUsecase), do.Lazy(NewBackupAccountUsecase),
	registry.Lazy(NewCacheUsecase), do.Lazy(NewCertUsecase), do.Lazy(NewCertAccountUsecase),
	registry.Lazy2(NewCertDNSUsecase), registry.Lazy2(NewContainerUsecase), registry.Lazy(NewContainerComposeUsecase),
	registry.Lazy2(NewContainerImageUsecase), registry.Lazy2(NewContainerNetworkUsecase), registry.Lazy2(NewContainerVolumeUsecase),
	registry.Lazy2(NewCronUsecase), do.Lazy(NewDatabaseUsecase), registry.Lazy(NewDatabaseRedisUsecase),
	registry.Lazy(NewDatabaseElasticsearchUsecase), do.Lazy(NewDatabaseServerUsecase), do.Lazy(NewDatabaseUserUsecase),
	do.Lazy(NewEnvironmentUsecase), registry.Lazy(NewLogUsecase), registry.Lazy2(NewMonitorUsecase),
	do.Lazy(NewProjectUsecase), registry.Lazy2(NewSafeUsecase), registry.Lazy2(NewScanEventUsecase),
	do.Lazy(NewSettingUsecase), registry.Lazy2(NewSSHUsecase), do.Lazy(NewTamperUsecase), registry.Lazy(NewTaskUsecase),
	do.Lazy(NewTemplateUsecase), do.Lazy(NewUserUsecase), registry.Lazy(NewUserPasskeyUsecase),
	registry.Lazy(NewUserTokenUsecase), do.Lazy(NewWebHookUsecase), do.Lazy(NewWebsiteUsecase),
	registry.Lazy(NewWebsiteStatUsecase),
)
