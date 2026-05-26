package paymenthandler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"iq-home/backend/internal/domain/payment"
)

// ── stub service ─────────────────────────────────────────────────────────────

type stubSvc struct {
	err error
}

func (s *stubSvc) ProcessWebhook(_ context.Context, _ payment.WebhookPayload) error {
	return s.err
}

// ── helpers ──────────────────────────────────────────────────────────────────

const testSecret = "supersecretkey"

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func doRequest(t *testing.T, h *Handler, body []byte, sig string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/payment/webhook", bytes.NewReader(body))
	req.Header.Set("X-Payment-Signature", sig)
	w := httptest.NewRecorder()
	h.Process(w, req)
	return w
}

func validBody(t *testing.T, orderID int64, status string) []byte {
	t.Helper()
	b, _ := json.Marshal(payment.WebhookPayload{OrderID: orderID, Status: status, TxID: "tx1"})
	return b
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestProcess_MissingSignature_401(t *testing.T) {
	h := New(&stubSvc{}, testSecret)
	body := validBody(t, 1, "success")
	w := doRequest(t, h, body, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestProcess_WrongSignature_401(t *testing.T) {
	h := New(&stubSvc{}, testSecret)
	body := validBody(t, 1, "success")
	w := doRequest(t, h, body, "deadbeef")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestProcess_EmptySecret_FailsClosed_401(t *testing.T) {
	h := New(&stubSvc{}, "") // no secret configured
	body := validBody(t, 1, "success")
	w := doRequest(t, h, body, sign("", body)) // attacker signs with empty secret
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestProcess_InvalidJSON_400(t *testing.T) {
	h := New(&stubSvc{}, testSecret)
	body := []byte(`not json`)
	w := doRequest(t, h, body, sign(testSecret, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestProcess_UnknownStatus_400(t *testing.T) {
	h := New(&stubSvc{err: payment.ErrUnknownStatus}, testSecret)
	body := validBody(t, 1, "pending")
	w := doRequest(t, h, body, sign(testSecret, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestProcess_OrderNotFound_404(t *testing.T) {
	h := New(&stubSvc{err: payment.ErrOrderNotFound}, testSecret)
	body := validBody(t, 999, "success")
	w := doRequest(t, h, body, sign(testSecret, body))
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestProcess_AlreadyHandled_200(t *testing.T) {
	h := New(&stubSvc{err: payment.ErrAlreadyHandled}, testSecret)
	body := validBody(t, 1, "success")
	w := doRequest(t, h, body, sign(testSecret, body))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestProcess_ValidSuccess_200(t *testing.T) {
	h := New(&stubSvc{}, testSecret)
	body := validBody(t, 1, "success")
	w := doRequest(t, h, body, sign(testSecret, body))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}
