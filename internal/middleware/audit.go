package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// AdminAudit logs every admin action with method, path, status and duration.
// Place after BasicAuth so only authenticated requests reach this middleware.
func AdminAudit(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			log.Warn("admin action",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"ip", r.RemoteAddr,
				"duration", time.Since(start).String(),
			)
		})
	}
}
