package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer 捕获 panic，记录日志并返回 JSON 错误响应
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				slog.Error(fmt.Sprintf("%v", rvr), slog.String("stack", string(debug.Stack())))

				// WebSocket 等升级连接无法写入 HTTP 响应
				if r.Header.Get("Connection") == "Upgrade" {
					return
				}

				Abort(w, http.StatusInternalServerError, fmt.Sprintf("%v", rvr))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
