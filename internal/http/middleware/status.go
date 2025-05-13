package middleware

import (
	"net/http"

	"github.com/go-rat/chix"
	"github.com/leonelquinteros/gotext"

	"github.com/tnb-labs/panel/internal/app"
)

// Status 检查程序状态
func Status(t *gotext.Locale) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch app.Status {
			case app.StatusUpgrade:
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusServiceUnavailable)
				render.JSON(chix.M{
					"message": t.Get("panel is upgrading, please refresh later"),
				})
				return
			case app.StatusMaintain:
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusServiceUnavailable)
				render.JSON(chix.M{
					"message": t.Get("panel is maintaining, please refresh later"),
				})
				return
			case app.StatusClosed:
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusForbidden)
				render.JSON(chix.M{
					"message": t.Get("panel is closed"),
				})
				return
			case app.StatusFailed:
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusInternalServerError)
				render.JSON(chix.M{
					"message": t.Get("panel run error, please check or contact support"),
				})
				return
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
