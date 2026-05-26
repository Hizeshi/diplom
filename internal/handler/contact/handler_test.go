package contacthandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	contacthandler "iq-home/backend/internal/handler/contact"
	"iq-home/backend/internal/domain/contact"
)

// ─── Mock ─────────────────────────────────────────────────────────────────────

type mockService struct {
	err          error
	lastRequest  contact.CreateRequest
}

func (m *mockService) Create(_ context.Context, req contact.CreateRequest) error {
	m.lastRequest = req
	return m.err
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestCreate_ValidRequest(t *testing.T) {
	svc := &mockService{}
	h := contacthandler.New(svc)

	body := `{"name":"Алибек","email":"ali@test.com","message":"Хочу узнать про товар"}`
	r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "true") {
		t.Errorf("expected success in body, got: %s", w.Body.String())
	}
	if svc.lastRequest.Email != "ali@test.com" {
		t.Errorf("expected email to reach service, got: %s", svc.lastRequest.Email)
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	h := contacthandler.New(&mockService{})

	body := `{"name":"Алибек","email":"not-email","message":"Хочу узнать про товар"}`
	r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"no name", `{"email":"a@b.com","message":"Хочу узнать про товар"}`},
		{"no email", `{"name":"Алибек","message":"Хочу узнать про товар"}`},
		{"no message", `{"name":"Алибек","email":"a@b.com"}`},
		{"empty body", `{}`},
	}

	h := contacthandler.New(&mockService{})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Create(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := contacthandler.New(&mockService{})

	r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(`{invalid`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_ServiceError_Returns500(t *testing.T) {
	svc := &mockService{err: errors.New("db connection lost")}
	h := contacthandler.New(svc)

	body := `{"name":"Алибек","email":"ali@test.com","message":"Хочу узнать про товар"}`
	r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestCreate_MessageTooShort(t *testing.T) {
	h := contacthandler.New(&mockService{})

	body := `{"name":"Алибек","email":"ali@test.com","message":"ok"}`
	r := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short message, got %d", w.Code)
	}
}
