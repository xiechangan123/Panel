package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gookit/validate"
	"github.com/knadh/koanf/v2"
	"github.com/robfig/cron/v3"

	"github.com/tnborg/panel/pkg/queue"
)

type Ace struct {
	conf     *koanf.Koanf
	router   *fiber.App
	migrator *gormigrate.Gormigrate
	cron     *cron.Cron
	queue    *queue.Queue
}

func NewWeb(conf *koanf.Koanf, router *fiber.App, migrator *gormigrate.Gormigrate, cron *cron.Cron, queue *queue.Queue, _ *validate.Validation) *Ace {
	return &Ace{
		conf:     conf,
		router:   router,
		migrator: migrator,
		cron:     cron,
		queue:    queue,
	}
}

func (r *Ace) Run() error {
	// migrate database
	if err := r.migrator.Migrate(); err != nil {
		return err
	}
	fmt.Println("[DB] database migrated")

	// start cron scheduler
	r.cron.Start()
	fmt.Println("[CRON] cron scheduler started")

	// start queue
	r.queue.Run(context.TODO())

	// run http server
	config := fiber.ListenConfig{
		ListenerNetwork:       fiber.NetworkTCP,
		EnablePrefork:         r.conf.Bool("http.prefork"),
		EnablePrintRoutes:     r.conf.Bool("http.debug"),
		DisableStartupMessage: !r.conf.Bool("http.debug"),
	}
	if r.conf.Bool("http.tls") {
		config.CertFile = filepath.Join(Root, "panel/storage/cert.pem")
		config.CertKeyFile = filepath.Join(Root, "panel/storage/cert.key")
		fmt.Println("[HTTP] listening and serving on port", r.conf.MustInt("http.port"), "with tls")
	} else {
		fmt.Println("[HTTP] listening and serving on port", r.conf.MustInt("http.port"))
	}

	return r.router.Listen(fmt.Sprintf(":%d", r.conf.MustInt("http.port")), config)
}

func (r *Ace) listenConfig() fiber.ListenConfig {
	// prefork not support dual stack
	return fiber.ListenConfig{
		ListenerNetwork:       fiber.NetworkTCP,
		EnablePrefork:         r.conf.Bool("http.prefork"),
		EnablePrintRoutes:     r.conf.Bool("http.debug"),
		DisableStartupMessage: !r.conf.Bool("http.debug"),
	}
}
