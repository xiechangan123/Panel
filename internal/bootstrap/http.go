package bootstrap

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/bddjr/hlfhr"
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/http/middleware"
	"github.com/acepanel/panel/internal/route"
	"github.com/acepanel/panel/pkg/config"
)

func NewRouter(t *gotext.Locale, middlewares *middleware.Middlewares, http *route.Http, ws *route.Ws) (*chi.Mux, error) {
	r := chi.NewRouter()

	// add middleware
	r.Use(middlewares.Globals(t, r)...)
	// add http route
	http.Register(r)
	// add ws route
	ws.Register(r)

	return r, nil
}

func NewHttp(conf *config.Config, mux *chi.Mux) (*hlfhr.Server, error) {
	srv := hlfhr.New(&http.Server{
		Addr:           fmt.Sprintf(":%d", conf.HTTP.Port),
		Handler:        mux,
		MaxHeaderBytes: 4 << 20,
	})
	srv.Listen80RedirectTo443 = true

	if conf.HTTP.TLS {
		srv.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return srv, nil
}
