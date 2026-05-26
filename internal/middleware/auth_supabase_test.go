package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"iq-home/backend/internal/middleware"
)

// makeGetUser returns a GetUserFunc that counts calls and returns fixed id/email.
func makeGetUser(id, email string, calls *atomic.Int64) middleware.GetUserFunc {
	return func(_ context.Context, _ string) (string, string, error) {
		calls.Add(1)
		return id, email, nil
	}
}

func TestSupabaseAuth_ValidToken(t *testing.T) {
	var calls atomic.Int64
	h := middleware.SupabaseAuth(makeGetUser("user-1", "a@b.com", &calls))(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSupabaseAuth_MissingToken_Returns401(t *testing.T) {
	var calls atomic.Int64
	h := middleware.SupabaseAuth(makeGetUser("user-1", "a@b.com", &calls))(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if calls.Load() != 0 {
		t.Error("getUser should not be called when token is missing")
	}
}

func TestSupabaseAuth_InvalidToken_Returns401(t *testing.T) {
	getUser := func(_ context.Context, _ string) (string, string, error) {
		return "", "", nil // empty id = unauthorized
	}
	h := middleware.SupabaseAuth(getUser)(ok)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// TestSupabaseAuth_CachesToken verifies that repeated requests with the same token
// result in only ONE call to getUser (all subsequent hits come from cache).
func TestSupabaseAuth_CachesToken(t *testing.T) {
	var calls atomic.Int64
	h := middleware.SupabaseAuth(makeGetUser("user-1", "a@b.com", &calls))(ok)

	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer same-token")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	if calls.Load() != 1 {
		t.Errorf("expected getUser to be called exactly once, got %d calls", calls.Load())
	}
}

// TestSupabaseAuth_DifferentTokens verifies each unique token is validated separately.
func TestSupabaseAuth_DifferentTokens(t *testing.T) {
	var calls atomic.Int64
	h := middleware.SupabaseAuth(makeGetUser("user-1", "a@b.com", &calls))(ok)

	tokens := []string{"token-a", "token-b", "token-c"}
	for _, tok := range tokens {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}

	if calls.Load() != int64(len(tokens)) {
		t.Errorf("expected %d getUser calls, got %d", len(tokens), calls.Load())
	}
}

// TestSupabaseAuth_UserInContext verifies the user is accessible via UserFromContext.
func TestSupabaseAuth_UserInContext(t *testing.T) {
	getUser := func(_ context.Context, _ string) (string, string, error) {
		return "user-42", "test@mail.com", nil
	}

	var gotUser middleware.SupabaseUser
	checkCtx := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser, _ = middleware.UserFromContext(r.Context())
	})

	h := middleware.SupabaseAuth(getUser)(checkCtx)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer tok")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if gotUser.ID != "user-42" {
		t.Errorf("expected user id=user-42, got %q", gotUser.ID)
	}
	if gotUser.Email != "test@mail.com" {
		t.Errorf("expected email=test@mail.com, got %q", gotUser.Email)
	}
}
