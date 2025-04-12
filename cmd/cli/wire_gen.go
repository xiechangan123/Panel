// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/tnb-labs/panel/internal/app"
	"github.com/tnb-labs/panel/internal/apps/benchmark"
	"github.com/tnb-labs/panel/internal/apps/docker"
	"github.com/tnb-labs/panel/internal/apps/fail2ban"
	"github.com/tnb-labs/panel/internal/apps/frp"
	"github.com/tnb-labs/panel/internal/apps/gitea"
	"github.com/tnb-labs/panel/internal/apps/memcached"
	"github.com/tnb-labs/panel/internal/apps/minio"
	"github.com/tnb-labs/panel/internal/apps/mysql"
	"github.com/tnb-labs/panel/internal/apps/nginx"
	"github.com/tnb-labs/panel/internal/apps/php74"
	"github.com/tnb-labs/panel/internal/apps/php80"
	"github.com/tnb-labs/panel/internal/apps/php81"
	"github.com/tnb-labs/panel/internal/apps/php82"
	"github.com/tnb-labs/panel/internal/apps/php83"
	"github.com/tnb-labs/panel/internal/apps/php84"
	"github.com/tnb-labs/panel/internal/apps/phpmyadmin"
	"github.com/tnb-labs/panel/internal/apps/podman"
	"github.com/tnb-labs/panel/internal/apps/postgresql"
	"github.com/tnb-labs/panel/internal/apps/pureftpd"
	"github.com/tnb-labs/panel/internal/apps/redis"
	"github.com/tnb-labs/panel/internal/apps/rsync"
	"github.com/tnb-labs/panel/internal/apps/s3fs"
	"github.com/tnb-labs/panel/internal/apps/supervisor"
	"github.com/tnb-labs/panel/internal/apps/toolbox"
	"github.com/tnb-labs/panel/internal/bootstrap"
	"github.com/tnb-labs/panel/internal/data"
	"github.com/tnb-labs/panel/internal/route"
	"github.com/tnb-labs/panel/internal/service"
)

import (
	_ "time/tzdata"
)

// Injectors from wire.go:

// initCli init command line.
func initCli() (*app.Cli, error) {
	koanf, err := bootstrap.NewConf()
	if err != nil {
		return nil, err
	}
	locale, err := bootstrap.NewT(koanf)
	if err != nil {
		return nil, err
	}
	logger := bootstrap.NewLog(koanf)
	db, err := bootstrap.NewDB(koanf, logger)
	if err != nil {
		return nil, err
	}
	cacheRepo := data.NewCacheRepo(db)
	queue := bootstrap.NewQueue()
	taskRepo := data.NewTaskRepo(locale, db, logger, queue)
	appRepo := data.NewAppRepo(locale, db, cacheRepo, taskRepo)
	userRepo := data.NewUserRepo(locale, db)
	settingRepo := data.NewSettingRepo(locale, db, koanf, taskRepo)
	databaseServerRepo := data.NewDatabaseServerRepo(locale, db, logger)
	databaseUserRepo := data.NewDatabaseUserRepo(locale, db, databaseServerRepo)
	databaseRepo := data.NewDatabaseRepo(locale, db, databaseServerRepo, databaseUserRepo)
	certRepo := data.NewCertRepo(locale, db, logger)
	certAccountRepo := data.NewCertAccountRepo(locale, db, userRepo, logger)
	websiteRepo := data.NewWebsiteRepo(db, cacheRepo, databaseRepo, databaseServerRepo, databaseUserRepo, certRepo, certAccountRepo)
	backupRepo := data.NewBackupRepo(locale, db, settingRepo, websiteRepo)
	cliService := service.NewCliService(locale, koanf, db, appRepo, cacheRepo, userRepo, settingRepo, backupRepo, websiteRepo, databaseServerRepo)
	cli := route.NewCli(cliService)
	command := bootstrap.NewCli(locale, cli)
	gormigrate := bootstrap.NewMigrate(db)
	benchmarkApp := benchmark.NewApp(locale)
	dockerApp := docker.NewApp()
	fail2banApp := fail2ban.NewApp(locale, websiteRepo)
	frpApp := frp.NewApp()
	giteaApp := gitea.NewApp()
	memcachedApp := memcached.NewApp(locale)
	minioApp := minio.NewApp()
	mysqlApp := mysql.NewApp(locale, settingRepo)
	nginxApp := nginx.NewApp(locale)
	php74App := php74.NewApp(locale, taskRepo)
	php80App := php80.NewApp(locale, taskRepo)
	php81App := php81.NewApp(locale, taskRepo)
	php82App := php82.NewApp(locale, taskRepo)
	php83App := php83.NewApp(locale, taskRepo)
	php84App := php84.NewApp(locale, taskRepo)
	phpmyadminApp := phpmyadmin.NewApp(locale)
	podmanApp := podman.NewApp()
	postgresqlApp := postgresql.NewApp(locale)
	pureftpdApp := pureftpd.NewApp(locale)
	redisApp := redis.NewApp(locale)
	rsyncApp := rsync.NewApp(locale)
	s3fsApp := s3fs.NewApp(locale, settingRepo)
	supervisorApp := supervisor.NewApp(locale)
	toolboxApp := toolbox.NewApp(locale)
	loader := bootstrap.NewLoader(benchmarkApp, dockerApp, fail2banApp, frpApp, giteaApp, memcachedApp, minioApp, mysqlApp, nginxApp, php74App, php80App, php81App, php82App, php83App, php84App, phpmyadminApp, podmanApp, postgresqlApp, pureftpdApp, redisApp, rsyncApp, s3fsApp, supervisorApp, toolboxApp)
	appCli := app.NewCli(command, gormigrate, loader)
	return appCli, nil
}
