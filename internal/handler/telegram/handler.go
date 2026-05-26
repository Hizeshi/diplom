package telegramhandler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"

	telegramsvc "iq-home/backend/internal/service/telegram"
	"iq-home/backend/pkg/respond"
)

type service interface {
	HandleUpdate(ctx context.Context, upd telegramsvc.Update) error
}

type Handler struct {
	svc     service
	secret  []byte
	enabled bool
}

func New(svc service, webhookSecret string) *Handler {
	return &Handler{
		svc:     svc,
		secret:  []byte(webhookSecret),
		enabled: svc != nil && webhookSecret != "",
	}
}

// POST /v1/telegram/webhook
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	// Fail-closed: if Telegram integration is not fully configured, return 503.
	if !h.enabled {
		http.Error(w, "telegram integration not configured", http.StatusServiceUnavailable)
		return
	}

	// Verify secret using constant-time comparison to prevent timing attacks.
	got := []byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
	if len(got) == 0 || subtle.ConstantTimeCompare(got, h.secret) != 1 {
		respond.Unauthorized(w)
		return
	}

	// Limit body size to prevent large-payload abuse.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var upd telegramsvc.Update
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}

	// Telegram expects a fast 200 OK; errors are logged by the service.
	_ = h.svc.HandleUpdate(r.Context(), upd)

	w.WriteHeader(http.StatusOK)
}
