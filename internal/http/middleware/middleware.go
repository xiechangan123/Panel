package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/google/wire"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"github.com/knadh/koanf/v2"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/sessions"
	sessionmiddleware "github.com/libtnb/sessions/middleware"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
)

var ProviderSet = wire.NewSet(NewMiddlewares)

type Middlewares struct {
	conf      *koanf.Koanf
	log       *slog.Logger
	session   *sessions.Manager
	appRepo   biz.AppRepo
	userToken biz.UserTokenRepo
}

func NewMiddlewares(conf *koanf.Koanf, session *sessions.Manager, appRepo biz.AppRepo, userToken biz.UserTokenRepo) *Middlewares {
	ljLogger := &lumberjack.Logger{
		Filename: filepath.Join(app.Root, "panel/storage/logs/http.log"),
		MaxSize:  10,
		MaxAge:   30,
		Compress: true,
	}

	return &Middlewares{
		conf:      conf,
		log:       slog.New(slog.NewJSONHandler(ljLogger, &slog.HandlerOptions{Level: slog.LevelInfo})),
		session:   session,
		appRepo:   appRepo,
		userToken: userToken,
	}
}

// Globals is a collection of global middleware that will be applied to every request.
func (r *Middlewares) Globals(t *gotext.Locale, mux *chi.Mux) []func(http.Handler) http.Handler {
	compressor := middleware.NewCompressor(5)
	compressor.SetEncoder("gzip", func(w io.Writer, level int) io.Writer {
		writer, _ := gzip.NewWriterLevel(w, level)
		return writer
	})
	compressor.SetEncoder("br", func(w io.Writer, level int) io.Writer {
		return brotli.NewWriterV2(w, 6)
	})
	compressor.SetEncoder("zstd", func(w io.Writer, level int) io.Writer {
		writer, _ := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
		return writer
	})

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
