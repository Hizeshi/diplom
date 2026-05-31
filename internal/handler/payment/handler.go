package paymenthandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"iq-home/backend/internal/domain/payment"
	"iq-home/backend/pkg/respond"
)

type service interface {
	ProcessWebhook(ctx context.Context, p payment.WebhookPayload) error
}

type Handler struct {
	svc    service
	secret []byte
}

func New(svc service, secret string) *Handler {
	return &Handler{svc: svc, secret: []byte(secret)}
}

// POST /api/payment/webhook
func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	// Read raw body so we can verify signature before decoding.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respond.BadRequest(w, "cannot read body")
		return
	}

	// Verify HMAC-SHA256 signature. Fail closed: no secret configured = reject all.
	sig := r.Header.Get("X-Payment-Signature")
	if !h.validSignature(sig, body) {
		respond.Unauthorized(w)
		return
	}

	var p payment.WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if p.ResolvedOrderID() == 0 {
		respond.BadRequest(w, "order_id is required")
		return
	}
	p.OrderID = p.ResolvedOrderID()

	if err := h.svc.ProcessWebhook(r.Context(), p); err != nil {
		switch {
		case errors.Is(err, payment.ErrUnknownStatus):
			respond.BadRequest(w, "unknown payment status")
		case errors.Is(err, payment.ErrOrderNotFound):
			respond.NotFound(w)
		case errors.Is(err, payment.ErrAlreadyHandled):
			// Idempotent: webhook was already processed successfully.
			respond.OK(w, map[string]bool{"success": true})
		default:
			respond.InternalError(w)
		}
		return
	}

	respond.OK(w, map[string]bool{"success": true})
}

// POST /api/payment/notify
// Called by l-xor-pay.vercel.app after payment attempt (no HMAC, rate-limited at router level).
// Body: { "orderId": 123, "status": "success" | "failed" }
func (h *Handler) Notify(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		respond.BadRequest(w, "cannot read body")
		return
	}

	var p payment.WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	p.OrderID = p.ResolvedOrderID()
	if p.OrderID == 0 {
		respond.BadRequest(w, "orderId is required")
		return
	}

	if err := h.svc.ProcessWebhook(r.Context(), p); err != nil {
		switch {
		case errors.Is(err, payment.ErrUnknownStatus):
			respond.BadRequest(w, "unknown payment status: "+p.Status)
		case errors.Is(err, payment.ErrOrderNotFound):
			respond.NotFound(w)
		case errors.Is(err, payment.ErrAlreadyHandled):
			respond.OK(w, map[string]bool{"success": true})
		default:
			slog.Error("payment notify error", "order_id", p.OrderID, "status", p.Status, "body", string(body), "err", err)
			respond.InternalError(w)
		}
		return
	}

	respond.OK(w, map[string]bool{"success": true})
}

// validSignature computes HMAC-SHA256(secret, body) and compares with sig
// using constant-time comparison to prevent timing attacks.
func (h *Handler) validSignature(sig string, body []byte) bool {
	if len(h.secret) == 0 {
		return false // fail closed: no secret → reject everything
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	sigBytes := []byte(sig)
	expBytes := []byte(expected)
	if len(sigBytes) != len(expBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(sigBytes, expBytes) == 1
}
