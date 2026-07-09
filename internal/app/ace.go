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
	"github.com/libtnb/cron"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/tlscert"
	"github.com/acepanel/panel/v3/pkg/types"
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

func NewAce(i do.Injector) (*Ace, error) {
	return &Ace{
		conf:     do.MustInvoke[*config.Config](i),
		router:   do.MustInvoke[*chi.Mux](i),
		server:   do.MustInvoke[*hlfhr.Server](i),
		reloader: do.MustInvoke[*tlscert.Reloader](i),
		migrator: do.MustInvoke[*gormigrate.Gormigrate](i),
		cron:     do.MustInvoke[*cron.Cron](i),
		runner:   do.MustInvoke[types.TaskRunner](i),
	}, nil
}

func (r *Ace) Run() error {
	// migrate database
	if err := r.migrator.Migrate(); err != nil {
		return err
	}
	fmt.Println("[DB] database migrated")

	// start cron scheduler
	if err := r.cron.Start(); err != nil {
		return err
	}
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
		if r.conf.HTTP.IsHTTPS() {
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
	cronCtx, cronCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = r.cron.Stop(cronCtx)
	cronCancel()
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
