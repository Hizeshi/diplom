package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"iq-home/backend/pkg/respond"
)

// ─── Trusted source context ──────────────────────────────────────────────────

type ctxTrustedKey struct{}

// MarkTrusted marks the request as coming from a trusted internal caller
// (internal token or Telegram webhook). Chat sessions created/accessed through
// trusted routes skip anonymous session-token enforcement.
func MarkTrusted(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxTrustedKey{}, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsTrusted reports whether the request context was marked as trusted.
func IsTrusted(ctx context.Context) bool {
	v, _ := ctx.Value(ctxTrustedKey{}).(bool)
	return v
}

// InternalToken validates the X-Internal-Token header using constant-time comparison.
// Fail-closed: if token is empty, all requests are rejected.
func InternalToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-Internal-Token")
			if token == "" || !constantTimeEqual(got, token) {
				respond.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BasicAuth validates Authorization: Basic ... header using constant-time comparison.
// Fail-closed: if username or password is empty, all requests are rejected.
func BasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := parseBasicAuth(r.Header.Get("Authorization"))
			uOK := ok && username != "" && constantTimeEqual(u, username)
			pOK := ok && password != "" && constantTimeEqual(p, password)
			if !uOK || !pOK {
				w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
				respond.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// constantTimeEqual compares two strings in constant time to prevent timing attacks.
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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
