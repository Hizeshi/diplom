package quotehandler

import (
	"context"
	"encoding/json"
	"net/http"

	"iq-home/backend/internal/domain/quote"
	"iq-home/backend/pkg/respond"
)

type service interface {
	Generate(ctx context.Context, req quote.Request) (*quote.Response, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/quotes
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req quote.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if len(req.Items) == 0 {
		respond.BadRequest(w, "items are required")
		return
	}

	resp, err := h.svc.Generate(r.Context(), req)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, resp)
}
