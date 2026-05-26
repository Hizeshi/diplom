package paymenthandler

import (
	"context"
	"encoding/json"
	"net/http"

	"iq-home/backend/internal/domain/payment"
	"iq-home/backend/pkg/respond"
)

type service interface {
	ProcessWebhook(ctx context.Context, p payment.WebhookPayload) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/payment/webhook
func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	var p payment.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if p.OrderID == 0 || p.Status == "" {
		respond.BadRequest(w, "order_id and status are required")
		return
	}
	if err := h.svc.ProcessWebhook(r.Context(), p); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}
