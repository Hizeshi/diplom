package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxRequestIDKey struct{}

const RequestIDHeader = "X-Request-ID"

// RequestID reads or generates a request ID and stores it in the context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), ctxRequestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request ID stored by the RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestIDKey{}).(string)
	return id
}
