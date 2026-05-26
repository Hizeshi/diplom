package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"iq-home/backend/pkg/respond"
)

// InternalToken validates the X-Internal-Token header.
func InternalToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Token") != token {
				respond.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BasicAuth validates Authorization: Basic ... header.
func BasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := parseBasicAuth(r.Header.Get("Authorization"))
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
				respond.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseBasicAuth(header string) (string, string, bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}
	u, p, ok := strings.Cut(string(decoded), ":")
	return u, p, ok
}
