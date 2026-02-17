package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bddjr/hlfhr"
	"github.com/go-chi/chi/v5"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gookit/validate"
	"github.com/robfig/cron/v3"

	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/tlscert"
	"github.com/acepanel/panel/pkg/types"
)

type Ace struct {
	conf     *config.Config
	router   *chi.Mux
	server   *hlfhr.Server
	reloader *tlscert.Reloader
	migrator *gormigrate.Gormigrate
	cron     *cron.Cron
	runner   types.TaskRunner
}

func NewAce(conf *config.Config, router *chi.Mux, server *hlfhr.Server, reloader *tlscert.Reloader, migrator *gormigrate.Gormigrate, cron *cron.Cron, runner types.TaskRunner, _ *validate.Validation) *Ace {
	return &Ace{
		conf:     conf,
		router:   router,
		server:   server,
		reloader: reloader,
		migrator: migrator,
		cron:     cron,
		runner:   runner,
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

	// create context for runner
	runnerCtx, runnerCancel := context.WithCancel(context.Background())
	defer runnerCancel()

	// start task runner
	r.runner.Run(runnerCtx)

	// setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// run http server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		fmt.Println("[HTTP] listening and serving on port", r.conf.HTTP.Port)
		if r.conf.HTTP.TLS {
			if err := r.server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		} else {
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
		fmt.Println("[APP] received signal:", sig)
	}

	// graceful shutdown
	fmt.Println("[APP] shutting down gracefully...")

	// stop cron scheduler
	ctx := r.cron.Stop()
	<-ctx.Done()
	fmt.Println("[CRON] cron scheduler stopped")

	// stop task runner
	runnerCancel()
	fmt.Println("[QUEUE] task runner stopped")

	// close certificate reloader
	if r.reloader != nil {
		if err := r.reloader.Close(); err != nil {
			fmt.Println("[TLS] certificate reloader close error:", err)
		}
	}

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
