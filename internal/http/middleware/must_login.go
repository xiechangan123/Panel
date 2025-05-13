package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-rat/chix"
	"github.com/go-rat/sessions"
	"github.com/go-rat/utils/str"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"
)

// MustLogin 确保已登录
func MustLogin(t *gotext.Locale, session *sessions.Manager) func(next http.Handler) http.Handler {
	// 白名单
	whiteList := []string{
		"/api/user/key",
		"/api/user/login",
		"/api/user/logout",
		"/api/user/isLogin",
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
				signature := strings.TrimPrefix(r.Header.Get("Authorization"), "HMAC-SHA256 ")

				// 步骤一：构造规范化请求
				body, err := io.ReadAll(r.Body)
				if err != nil {
					render := chix.NewRender(w)
					defer render.Release()
					render.Status(http.StatusInternalServerError)
					render.JSON(chix.M{
						"message": err.Error(),
					})
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
				canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s", r.Method, r.URL.Path, r.URL.Query().Encode(), str.SHA256(string(body)))

				// 步骤二：构造待签名字符串
				stringToSign := fmt.Sprintf("%s\n%d\n%s", "HMAC-SHA256", cast.ToInt64(r.Header.Get("X-Timestamp")), str.SHA256(canonicalRequest))

				// 步骤三：计算签名
				validSignature := hmacsha256(stringToSign, cast.ToString(sess.Get("api_secret")))

				// 步骤四：验证签名
				if subtle.ConstantTimeCompare([]byte(signature), []byte(validSignature)) != 1 {
					render := chix.NewRender(w)
					defer render.Release()
					render.Status(http.StatusUnauthorized)
					render.JSON(chix.M{
						"message": t.Get("invalid api signature"),
					})
					return
				}
				timestamp := cast.ToInt64(r.Header.Get("X-Timestamp"))
				if timestamp == 0 || timestamp < (time.Now().Unix()-60) {
					render := chix.NewRender(w)
					defer render.Release()
					render.Status(http.StatusUnauthorized)
					render.JSON(chix.M{
						"message": t.Get("api signature expired"),
					})
					return
				}

				// 步骤五：验证通过
				userID = 1
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

func hmacsha256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
