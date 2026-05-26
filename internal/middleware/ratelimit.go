package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"iq-home/backend/pkg/respond"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns a middleware that limits requests per IP using a token bucket.
// r — requests per second, burst — max burst size.
func RateLimit(r rate.Limit, burst int) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*ipLimiter)
	)

	// Clean up stale entries every minute.
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > 5*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		l, ok := limiters[ip]
		if !ok {
			l = &ipLimiter{limiter: rate.NewLimiter(r, burst)}
			limiters[ip] = l
		}
		l.lastSeen = time.Now()
		return l.limiter
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !getLimiter(ip).Allow() {
				respond.TooManyRequests(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
