package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRecovery_PanicReturns500(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestRecovery_NoPanicPassesThrough(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestRequestID_Generated(t *testing.T) {
	var gotID string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if gotID == "" {
		t.Fatal("expected request ID to be set in context")
	}
	if w.Header().Get(RequestIDHeader) != gotID {
		t.Fatal("response header should match context value")
	}
}

func TestRequestID_FromHeader(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "my-custom-id")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) != "my-custom-id" {
		t.Fatalf("expected custom ID to be echoed, got %q", w.Header().Get(RequestIDHeader))
	}
}
