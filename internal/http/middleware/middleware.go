package middleware

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/DeRuina/timberjack"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/google/wire"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/sessions"
	sessionmiddleware "github.com/libtnb/sessions/middleware"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/config"
)

var ProviderSet = wire.NewSet(NewMiddlewares)

type Middlewares struct {
	conf      *config.Config
	log       *slog.Logger
	session   *sessions.Manager
	appRepo   biz.AppRepo
	userToken biz.UserTokenRepo
}

func NewMiddlewares(conf *config.Config, session *sessions.Manager, appRepo biz.AppRepo, userToken biz.UserTokenRepo) *Middlewares {
	tjLogger := &timberjack.Logger{
		Filename:    filepath.Join(app.Root, "panel/storage/logs/http.log"),
		MaxSize:     10,
		MaxAge:      30,
		LocalTime:   true,
		RotateAt:    []string{"00:00"},
		FileMode:    0o600,
		Compression: "none",
	}

	return &Middlewares{
		conf:      conf,
		log:       slog.New(slog.NewJSONHandler(tjLogger, &slog.HandlerOptions{Level: slog.LevelInfo})),
		session:   session,
		appRepo:   appRepo,
		userToken: userToken,
	}
}

// Globals is a collection of global middleware that will be applied to every request.
func (r *Middlewares) Globals(t *gotext.Locale, mux *chi.Mux) []func(http.Handler) http.Handler {
	compressor := middleware.NewCompressor(6)

	return []func(http.Handler) http.Handler{
		middleware.Recoverer,
		httplog.RequestLogger(r.log, &httplog.Options{
			Level:             slog.LevelInfo,
			LogRequestHeaders: []string{"User-Agent"},
			Skip: func(req *http.Request, respStatus int) bool {
				return respStatus == 404 || respStatus == 405
			},
			LogRequestBody: func(req *http.Request) bool {
				return req.Header.Get("X-Debug-Request") == "1"
			},
			LogResponseBody: func(req *http.Request) bool {
				return req.Header.Get("X-Debug-Response") == "1"
			},
		}),
		compressor.Handler,
		sessionmiddleware.StartSession(r.session),
		Status(t),
		Entrance(t, r.conf, r.session),
		MustLogin(t, r.conf, r.session, r.userToken),
		MustInstall(t, r.appRepo),
	}
}
