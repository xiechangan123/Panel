package data

import (
	"github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy(NewAppRepo), do.Lazy(NewBackupRepo), do.Lazy(NewBackupAccountRepo),
	do.Lazy(NewCacheRepo), do.Lazy(NewCertRepo), do.Lazy(NewCertAccountRepo),
	do.Lazy(NewCertDNSRepo), do.Lazy(NewContainerRepo), do.Lazy(NewContainerComposeRepo),
	do.Lazy(NewContainerImageRepo), do.Lazy(NewContainerNetworkRepo), do.Lazy(NewContainerVolumeRepo),
	do.Lazy(NewCronRepo), do.Lazy(NewDatabaseRepo), do.Lazy(NewDatabaseRedisRepo),
	do.Lazy(NewDatabaseElasticsearchRepo), do.Lazy(NewDatabaseServerRepo), do.Lazy(NewDatabaseUserRepo),
	do.Lazy(NewEnvironmentRepo), do.Lazy(NewLogRepo), do.Lazy(NewMonitorRepo),
	do.Lazy(NewProjectRepo), do.Lazy(NewSafeRepo), do.Lazy(NewScanEventRepo),
	do.Lazy(NewSettingRepo), do.Lazy(NewSSHRepo), do.Lazy(NewTaskRepo),
	do.Lazy(NewTemplateRepo), do.Lazy(NewUserRepo), do.Lazy(NewUserPasskeyRepo),
	do.Lazy(NewUserTokenRepo), do.Lazy(NewWebHookRepo), do.Lazy(NewWebsiteRepo),
	do.Lazy(NewWebsiteStatRepo),
)
