package middleware

import (
	"context"
	"net/http"
	"strings"

	"iq-home/backend/pkg/respond"
)

type ctxKey int

const CtxUserID ctxKey = iota

// SupabaseUser is stored in request context after successful JWT validation.
type SupabaseUser struct {
	ID    string
	Email string
}

// GetUserFunc is the function the middleware calls to validate a Bearer token.
// Pass sb.GetUser wrapped in a closure so we don't import supabase here.
type GetUserFunc func(ctx context.Context, token string) (id, email string, err error)

// SupabaseAuth validates the Bearer token and stores the user in context.
func SupabaseAuth(getUser GetUserFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				respond.Unauthorized(w)
				return
			}

			id, email, err := getUser(r.Context(), token)
			if err != nil || id == "" {
				respond.Unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, SupabaseUser{ID: id, Email: email})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the authenticated user from request context.
func UserFromContext(ctx context.Context) (SupabaseUser, bool) {
	u, ok := ctx.Value(CtxUserID).(SupabaseUser)
	return u, ok
}

func extractBearer(header string) string {
	token, _ := strings.CutPrefix(header, "Bearer ")
	return token
}
