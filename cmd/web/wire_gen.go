// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/TheTNB/panel/internal/app"
	"github.com/TheTNB/panel/internal/apps"
	"github.com/TheTNB/panel/internal/apps/benchmark"
	"github.com/TheTNB/panel/internal/apps/docker"
	"github.com/TheTNB/panel/internal/apps/fail2ban"
	"github.com/TheTNB/panel/internal/apps/frp"
	"github.com/TheTNB/panel/internal/apps/gitea"
	"github.com/TheTNB/panel/internal/apps/memcached"
	"github.com/TheTNB/panel/internal/apps/mysql"
	"github.com/TheTNB/panel/internal/apps/nginx"
	"github.com/TheTNB/panel/internal/apps/php"
	"github.com/TheTNB/panel/internal/apps/phpmyadmin"
	"github.com/TheTNB/panel/internal/apps/podman"
	"github.com/TheTNB/panel/internal/apps/postgresql"
	"github.com/TheTNB/panel/internal/apps/pureftpd"
	"github.com/TheTNB/panel/internal/apps/redis"
	"github.com/TheTNB/panel/internal/apps/rsync"
	"github.com/TheTNB/panel/internal/apps/s3fs"
	"github.com/TheTNB/panel/internal/apps/supervisor"
	"github.com/TheTNB/panel/internal/apps/toolbox"
	"github.com/TheTNB/panel/internal/bootstrap"
	"github.com/TheTNB/panel/internal/data"
	"github.com/TheTNB/panel/internal/http/middleware"
	"github.com/TheTNB/panel/internal/job"
	"github.com/TheTNB/panel/internal/route"
	"github.com/TheTNB/panel/internal/service"
)

import (
	_ "time/tzdata"
)

// Injectors from wire.go:

// initWeb init application.
func initWeb() (*app.Web, error) {
	koanf, err := bootstrap.NewConf()
	if err != nil {
		return nil, err
	}
	logger := bootstrap.NewLog(koanf)
	db, err := bootstrap.NewDB(koanf, logger)
	if err != nil {
		return nil, err
	}
	manager, err := bootstrap.NewSession(koanf, db)
	if err != nil {
		return nil, err
	}
	cacheRepo := data.NewCacheRepo(db)
	queue := bootstrap.NewQueue()
	taskRepo := data.NewTaskRepo(db, logger, queue)
	appRepo := data.NewAppRepo(db, cacheRepo, taskRepo)
	middlewares := middleware.NewMiddlewares(koanf, logger, manager, appRepo)
	userRepo := data.NewUserRepo(db)
	userService := service.NewUserService(koanf, manager, userRepo)
	databaseServerRepo := data.NewDatabaseServerRepo(db, logger)
	databaseUserRepo := data.NewDatabaseUserRepo(db, databaseServerRepo)
	databaseRepo := data.NewDatabaseRepo(db, databaseServerRepo, databaseUserRepo)
	certRepo := data.NewCertRepo(db)
	certAccountRepo := data.NewCertAccountRepo(db, userRepo)
	websiteRepo := data.NewWebsiteRepo(db, cacheRepo, databaseRepo, databaseServerRepo, databaseUserRepo, certRepo, certAccountRepo)
	settingRepo := data.NewSettingRepo(db, koanf, taskRepo)
	cronRepo := data.NewCronRepo(db)
	backupRepo := data.NewBackupRepo(db, settingRepo, websiteRepo)
	dashboardService := service.NewDashboardService(koanf, taskRepo, websiteRepo, appRepo, settingRepo, cronRepo, backupRepo)
	taskService := service.NewTaskService(taskRepo)
	websiteService := service.NewWebsiteService(websiteRepo, settingRepo)
	databaseService := service.NewDatabaseService(databaseRepo)
	databaseServerService := service.NewDatabaseServerService(databaseServerRepo)
	databaseUserService := service.NewDatabaseUserService(databaseUserRepo)
	backupService := service.NewBackupService(backupRepo)
	certService := service.NewCertService(certRepo)
	certDNSRepo := data.NewCertDNSRepo(db)
	certDNSService := service.NewCertDNSService(certDNSRepo)
	certAccountService := service.NewCertAccountService(certAccountRepo)
	appService := service.NewAppService(appRepo, cacheRepo, settingRepo)
	cronService := service.NewCronService(cronRepo)
	processService := service.NewProcessService()
	safeRepo := data.NewSafeRepo()
	safeService := service.NewSafeService(safeRepo)
	firewallService := service.NewFirewallService()
	sshRepo := data.NewSSHRepo(db)
	sshService := service.NewSSHService(sshRepo)
	containerRepo := data.NewContainerRepo()
	containerService := service.NewContainerService(containerRepo)
	containerNetworkRepo := data.NewContainerNetworkRepo()
	containerNetworkService := service.NewContainerNetworkService(containerNetworkRepo)
	containerImageRepo := data.NewContainerImageRepo()
	containerImageService := service.NewContainerImageService(containerImageRepo)
	containerVolumeRepo := data.NewContainerVolumeRepo()
	containerVolumeService := service.NewContainerVolumeService(containerVolumeRepo)
	fileService := service.NewFileService(taskRepo)
	monitorRepo := data.NewMonitorRepo(db, settingRepo)
	monitorService := service.NewMonitorService(settingRepo, monitorRepo)
	settingService := service.NewSettingService(settingRepo)
	systemctlService := service.NewSystemctlService()
	benchmarkApp := benchmark.NewApp()
	dockerApp := docker.NewApp()
	fail2banApp := fail2ban.NewApp(websiteRepo)
	frpApp := frp.NewApp()
	giteaApp := gitea.NewApp()
	memcachedApp := memcached.NewApp()
	mysqlApp := mysql.NewApp(settingRepo)
	nginxApp := nginx.NewApp()
	phpApp := php.NewApp(taskRepo)
	phpmyadminApp := phpmyadmin.NewApp()
	podmanApp := podman.NewApp()
	postgresqlApp := postgresql.NewApp()
	pureftpdApp := pureftpd.NewApp()
	redisApp := redis.NewApp()
	rsyncApp := rsync.NewApp()
	s3fsApp := s3fs.NewApp(settingRepo)
	supervisorApp := supervisor.NewApp()
	toolboxApp := toolbox.NewApp()
	loader := apps.NewLoader(benchmarkApp, dockerApp, fail2banApp, frpApp, giteaApp, memcachedApp, mysqlApp, nginxApp, phpApp, phpmyadminApp, podmanApp, postgresqlApp, pureftpdApp, redisApp, rsyncApp, s3fsApp, supervisorApp, toolboxApp)
	http := route.NewHttp(userService, dashboardService, taskService, websiteService, databaseService, databaseServerService, databaseUserService, backupService, certService, certDNSService, certAccountService, appService, cronService, processService, safeService, firewallService, sshService, containerService, containerNetworkService, containerImageService, containerVolumeService, fileService, monitorService, settingService, systemctlService, loader)
	wsService := service.NewWsService(koanf, sshRepo)
	ws := route.NewWs(wsService)
	mux, err := bootstrap.NewRouter(middlewares, http, ws)
	if err != nil {
		return nil, err
	}
	server, err := bootstrap.NewHttp(koanf, mux)
	if err != nil {
		return nil, err
	}
	gormigrate := bootstrap.NewMigrate(db)
	jobs := job.NewJobs(db, logger, settingRepo, certRepo, backupRepo, cacheRepo)
	cron, err := bootstrap.NewCron(koanf, logger, jobs)
	if err != nil {
		return nil, err
	}
	validation := bootstrap.NewValidator(db)
	web := app.NewWeb(koanf, mux, server, gormigrate, cron, validation)
	return web, nil
}