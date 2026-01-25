package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bddjr/hlfhr"
	"github.com/go-chi/chi/v5"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gookit/validate"
	"github.com/robfig/cron/v3"

	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/queue"
)

type Ace struct {
	conf     *config.Config
	router   *chi.Mux
	server   *hlfhr.Server
	migrator *gormigrate.Gormigrate
	cron     *cron.Cron
	queue    *queue.Queue
}

func NewAce(conf *config.Config, router *chi.Mux, server *hlfhr.Server, migrator *gormigrate.Gormigrate, cron *cron.Cron, queue *queue.Queue, _ *validate.Validation) *Ace {
	return &Ace{
		conf:     conf,
		router:   router,
		server:   server,
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

	// create context for queue
	queueCtx, queueCancel := context.WithCancel(context.Background())
	defer queueCancel()

	// start queue
	r.queue.Run(queueCtx)

	// setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// run http server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if r.conf.HTTP.TLS {
			cert := filepath.Join(Root, "panel/storage/cert.pem")
			key := filepath.Join(Root, "panel/storage/cert.key")
			fmt.Println("[HTTP] listening and serving on port", r.conf.HTTP.Port, "with tls")
			if err := r.server.ListenAndServeTLS(cert, key); !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		} else {
			fmt.Println("[HTTP] listening and serving on port", r.conf.HTTP.Port)
			if err := r.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		}
		close(serverErr)
	}()

	// wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case sig := <-quit:
		fmt.Println("\n[APP] received signal:", sig)
	}

	// graceful shutdown
	fmt.Println("[APP] shutting down gracefully...")

	// stop cron scheduler
	ctx := r.cron.Stop()
	<-ctx.Done()
	fmt.Println("[CRON] cron scheduler stopped")

	// stop queue
	queueCancel()
	fmt.Println("[QUEUE] queue stopped")

	// shutdown http server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := r.server.Shutdown(shutdownCtx); err != nil {
		fmt.Println("[HTTP] server shutdown error:", err)
		return err
	}
	fmt.Println("[HTTP] server stopped")

	fmt.Println("[APP] shutdown complete")
	return nil
}
