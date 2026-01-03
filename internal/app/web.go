package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/bddjr/hlfhr"
	"github.com/go-chi/chi/v5"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gookit/validate"
	"github.com/robfig/cron/v3"

	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/queue"
)

type Web struct {
	conf     *config.Config
	router   *chi.Mux
	server   *hlfhr.Server
	migrator *gormigrate.Gormigrate
	cron     *cron.Cron
	queue    *queue.Queue
}

func NewWeb(conf *config.Config, router *chi.Mux, server *hlfhr.Server, migrator *gormigrate.Gormigrate, cron *cron.Cron, queue *queue.Queue, _ *validate.Validation) *Web {
	return &Web{
		conf:     conf,
		router:   router,
		server:   server,
		migrator: migrator,
		cron:     cron,
		queue:    queue,
	}
}

func (r *Web) Run() error {
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
	if r.conf.HTTP.TLS {
		cert := filepath.Join(Root, "panel/storage/cert.pem")
		key := filepath.Join(Root, "panel/storage/cert.key")
		fmt.Println("[HTTP] listening and serving on port", r.conf.HTTP.Port, "with tls")
		if err := r.server.ListenAndServeTLS(cert, key); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	} else {
		fmt.Println("[HTTP] listening and serving on port", r.conf.HTTP.Port)
		if err := r.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	return nil
}
