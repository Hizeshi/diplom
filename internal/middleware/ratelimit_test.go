package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"

	"iq-home/backend/internal/middleware"
)

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	// burst=5, rate=10 — first 5 requests must pass
	h := middleware.RateLimit(rate.Limit(10), 5)(ok)

	for i := 1; i <= 5; i++ {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	// burst=3 — 4th request from same IP must be blocked
	h := middleware.RateLimit(rate.Limit(0.001), 3)(ok)

	ip := "10.0.0.1:9999"
	var lastCode int
	for i := 1; i <= 10; i++ {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = ip
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		lastCode = w.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst exhausted, got %d", lastCode)
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	// burst=2 — each IP gets its own bucket
	h := middleware.RateLimit(rate.Limit(0.001), 2)(ok)

	ips := []string{"1.1.1.1:1000", "2.2.2.2:2000", "3.3.3.3:3000"}
	for _, ip := range ips {
		for i := 1; i <= 2; i++ {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.RemoteAddr = ip
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("ip %s request %d: expected 200, got %d", ip, i, w.Code)
			}
		}
	}
}

func TestRateLimit_Returns429WithRetryAfter(t *testing.T) {
	h := middleware.RateLimit(rate.Limit(0.001), 1)(ok)

	ip := "5.5.5.5:5000"
	// Exhaust the burst
	for i := 0; i < 3; i++ {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = ip
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}

	// Next one must be 429 with Retry-After
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = ip
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header in 429 response")
	}
}
