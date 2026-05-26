package contacthandler

import (
	"context"
	"net/http"

	"iq-home/backend/internal/domain/contact"
	"iq-home/backend/pkg/respond"
	"iq-home/backend/pkg/validate"
)

type service interface {
	Create(ctx context.Context, req contact.CreateRequest) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/contact
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req contact.CreateRequest
	if !validate.DecodeAndValidate(w, r, &req) {
		return
	}
	if err := h.svc.Create(r.Context(), req); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}
