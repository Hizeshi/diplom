package middleware_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"iq-home/backend/internal/middleware"
)

// ok is a simple handler that always returns 200.
var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ─── InternalToken ────────────────────────────────────────────────────────────

func TestInternalToken_Valid(t *testing.T) {
	h := middleware.InternalToken("secret123")(ok)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Internal-Token", "secret123")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInternalToken_Missing(t *testing.T) {
	h := middleware.InternalToken("secret123")(ok)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestInternalToken_Wrong(t *testing.T) {
	h := middleware.InternalToken("secret123")(ok)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Internal-Token", "wrong")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ─── BasicAuth ────────────────────────────────────────────────────────────────

func basicAuthHeader(user, pass string) string {
	creds := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + creds
}

func TestBasicAuth_Valid(t *testing.T) {
	h := middleware.BasicAuth("admin", "pass123")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "pass123"))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestBasicAuth_WrongPassword(t *testing.T) {
	h := middleware.BasicAuth("admin", "pass123")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", basicAuthHeader("admin", "wrong"))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuth_NoHeader(t *testing.T) {
	h := middleware.BasicAuth("admin", "pass123")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuth_WWWAuthenticateHeader(t *testing.T) {
	h := middleware.BasicAuth("admin", "pass")(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header in 401 response")
	}
}
