package compatibilityhandler

import (
	"context"
	"net/http"

	"iq-home/backend/internal/domain/compatibility"
	"iq-home/backend/internal/middleware"
	"iq-home/backend/pkg/respond"
)

type service interface {
	CheckCart(ctx context.Context, userID string) (*compatibility.Result, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/user/cart/compatibility
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respond.Unauthorized(w)
		return
	}

	result, err := h.svc.CheckCart(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}

	respond.OK(w, result)
}
