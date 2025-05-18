package middleware

import (
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-rat/chix"
	"github.com/go-rat/sessions"
	"github.com/knadh/koanf/v2"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"

	"github.com/tnb-labs/panel/pkg/punycode"
)

// Entrance 确保通过正确的入口访问
func Entrance(t *gotext.Locale, conf *koanf.Koanf, session *sessions.Manager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := session.GetSession(r)
			if err != nil {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusInternalServerError)
				render.JSON(chix.M{
					"msg": err.Error(),
				})
				return
			}

			entrance := strings.TrimSuffix(conf.String("http.entrance"), "/")
			if entrance == "" {
				entrance = "/"
			}
			if !strings.HasPrefix(entrance, "/") {
				entrance = "/" + entrance
			}

			// 情况一：设置了绑定域名、IP、UA，且请求不符合要求，返回错误
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host
			}
			if strings.Contains(host, "xn--") {
				if decoded, err := punycode.DecodeDomain(host); err == nil {
					host = decoded
				}
			}
			if len(conf.Strings("http.bind_domain")) > 0 && !slices.Contains(conf.Strings("http.bind_domain"), host) {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusTeapot)
				render.JSON(chix.M{
					"msg": t.Get("invalid request domain: %s", r.Host),
				})
				return
			}
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if len(conf.Strings("http.bind_ip")) > 0 && !slices.Contains(conf.Strings("http.bind_ip"), ip) {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusTeapot)
				render.JSON(chix.M{
					"msg": t.Get("invalid request ip: %s", ip),
				})
				return
			}
			if len(conf.Strings("http.bind_ua")) > 0 && !slices.Contains(conf.Strings("http.bind_ua"), r.UserAgent()) {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusTeapot)
				render.JSON(chix.M{
					"msg": t.Get("invalid request user agent: %s", r.UserAgent()),
				})
				return
			}

			// 情况二：请求路径与入口路径相同，标记通过验证并重定向到登录页面
			if strings.TrimSuffix(r.URL.Path, "/") == entrance {
				sess.Put("verify_entrance", true)
				render := chix.NewRender(w, r)
				defer render.Release()
				render.Redirect("/login")
				return
			}

			// 情况三：通过APIKey+入口路径访问，重写请求路径并跳过验证
			if strings.HasPrefix(r.URL.Path, entrance) && r.Header.Get("Authorization") != "" {
				// 只在设置了入口路径的情况下，才进行重写
				if entrance != "/" {
					if rctx := chi.RouteContext(r.Context()); rctx != nil {
						rctx.RoutePath = strings.TrimPrefix(rctx.RoutePath, entrance)
					}
					r.URL.Path = strings.TrimPrefix(r.URL.Path, entrance)
				}
				next.ServeHTTP(w, r)
				return
			}

			// 情况三：非调试模式且未通过验证的请求，返回错误
			if !conf.Bool("app.debug") &&
				!cast.ToBool(sess.Get("verify_entrance", false)) &&
				r.URL.Path != "/robots.txt" {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusTeapot)
				render.JSON(chix.M{
					"msg": t.Get("invalid access entrance"),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
