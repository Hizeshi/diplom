package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"iq-home/backend/pkg/respond"
)

// Recovery catches panics, logs the stack trace, and returns a 500 response.
// The server stays alive after a panic in any handler.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic recovered",
						"panic", v,
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					respond.InternalError(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
