package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/go-rat/chix"
	"github.com/go-rat/sessions"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"

	"github.com/tnb-labs/panel/internal/biz"
)

// MustLogin 确保已登录
func MustLogin(t *gotext.Locale, session *sessions.Manager, userToken biz.UserTokenRepo) func(next http.Handler) http.Handler {
	// 白名单
	whiteList := []string{
		"/api/user/key",
		"/api/user/login",
		"/api/user/logout",
		"/api/user/is_login",
		"/api/user/is_2fa",
		"/api/dashboard/panel",
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := session.GetSession(r)
			if err != nil {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusInternalServerError)
				render.JSON(chix.M{
					"message": err.Error(),
				})
				return
			}

			// 对白名单和非 API 请求放行
			if slices.Contains(whiteList, r.URL.Path) || !strings.HasPrefix(r.URL.Path, "/api") {
				next.ServeHTTP(w, r)
				return
			}

			userID := uint(0)
			if r.Header.Get("Authorization") != "" {
				// API 请求验证
				if userID, err = userToken.ValidateReq(r); err != nil {
					render := chix.NewRender(w)
					defer render.Release()
					render.Status(http.StatusUnauthorized)
					render.JSON(chix.M{
						"message": err.Error(),
					})
					return
				}
			} else {
				if sess.Missing("user_id") {
					render := chix.NewRender(w)
					defer render.Release()
					render.Status(http.StatusUnauthorized)
					render.JSON(chix.M{
						"message": t.Get("session expired, please login again"),
					})
					return
				}

				safeLogin := cast.ToBool(sess.Get("safe_login"))
				if safeLogin {
					safeClientHash := cast.ToString(sess.Get("safe_client"))
					ip, _, _ := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
					clientHash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))
					if safeClientHash != clientHash || safeClientHash == "" {
						render := chix.NewRender(w)
						defer render.Release()
						render.Status(http.StatusUnauthorized)
						render.JSON(chix.M{
							"message": t.Get("client ip/ua changed, please login again"),
						})
						return
					}
				}

				userID = cast.ToUint(sess.Get("user_id"))
			}

			if userID == 0 {
				render := chix.NewRender(w)
				defer render.Release()
				render.Status(http.StatusUnauthorized)
				render.JSON(chix.M{
					"message": t.Get("invalid user id, please login again"),
				})
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "user_id", userID)) // nolint:staticcheck
			next.ServeHTTP(w, r)
		})
	}
}
