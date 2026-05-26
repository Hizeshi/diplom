package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

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

// tokenTTL is how long a validated token is trusted without re-checking Supabase.
// Supabase JWTs expire after 1 hour by default, so 5 minutes is a safe trade-off
// between performance and detecting revocations.
const tokenTTL = 5 * time.Minute

type cacheEntry struct {
	user      SupabaseUser
	expiresAt time.Time
}

type tokenCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

func newTokenCache() *tokenCache {
	c := &tokenCache{entries: make(map[string]cacheEntry)}
	go c.cleanup()
	return c
}

func (c *tokenCache) get(token string) (SupabaseUser, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[token]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, token)
		return SupabaseUser{}, false
	}
	return e.user, true
}

func (c *tokenCache) set(token string, user SupabaseUser) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[token] = cacheEntry{user: user, expiresAt: time.Now().Add(tokenTTL)}
}

// cleanup removes expired entries every minute to prevent unbounded memory growth.
func (c *tokenCache) cleanup() {
	for range time.Tick(time.Minute) {
		c.mu.Lock()
		now := time.Now()
		for token, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, token)
			}
		}
		c.mu.Unlock()
	}
}

// SupabaseAuth validates the Bearer token and stores the user in context.
// Validated tokens are cached for tokenTTL to avoid a Supabase HTTP call on every request.
func SupabaseAuth(getUser GetUserFunc) func(http.Handler) http.Handler {
	cache := newTokenCache()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r.Header.Get("Authorization"))
			if token == "" {
				respond.Unauthorized(w)
				return
			}

			// Cache hit — no HTTP call needed.
			if user, ok := cache.get(token); ok {
				ctx := context.WithValue(r.Context(), CtxUserID, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Cache miss — validate with Supabase.
			id, email, err := getUser(r.Context(), token)
			if err != nil || id == "" {
				respond.Unauthorized(w)
				return
			}

			user := SupabaseUser{ID: id, Email: email}
			cache.set(token, user)

			ctx := context.WithValue(r.Context(), CtxUserID, user)
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
