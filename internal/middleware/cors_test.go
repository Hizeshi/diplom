package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"iq-home/backend/internal/middleware"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	h := middleware.CORS("https://iq-home.kz,https://www.iq-home.kz")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://iq-home.kz")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://iq-home.kz" {
		t.Errorf("expected exact origin, got %q", got)
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Error("expected Vary: Origin header")
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	h := middleware.CORS("https://iq-home.kz")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header for unknown origin, got %q", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	h := middleware.CORS("https://iq-home.kz")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	// Non-browser requests (curl, server-to-server) have no Origin — must pass through
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORS_WildcardAllowsAll(t *testing.T) {
	h := middleware.CORS("*")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected *, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	h := middleware.CORS("https://iq-home.kz")(ok)

	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.Header.Set("Origin", "https://iq-home.kz")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

func TestCORS_MultipleOriginsInConfig(t *testing.T) {
	h := middleware.CORS("https://iq-home.kz,https://www.iq-home.kz")(ok)

	for _, origin := range []string{"https://iq-home.kz", "https://www.iq-home.kz"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, r)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %s: expected %s, got %q", origin, origin, got)
		}
	}
}
