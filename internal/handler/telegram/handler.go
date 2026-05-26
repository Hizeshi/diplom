package telegramhandler

import (
	"context"
	"encoding/json"
	"net/http"

	telegramsvc "iq-home/backend/internal/service/telegram"
	"iq-home/backend/pkg/respond"
)

type service interface {
	HandleUpdate(ctx context.Context, upd telegramsvc.Update) error
}

type Handler struct {
	svc    service
	secret string
}

func New(svc service, webhookSecret string) *Handler {
	return &Handler{svc: svc, secret: webhookSecret}
}

// POST /v1/telegram/webhook
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != h.secret {
		respond.Unauthorized(w)
		return
	}

	var upd telegramsvc.Update
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}

	// Telegram expects a fast 200 OK; errors are swallowed so it doesn't retry.
	_ = h.svc.HandleUpdate(r.Context(), upd)

	w.WriteHeader(http.StatusOK)
}
