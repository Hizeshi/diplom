package describehandler

import (
	"context"
	"net/http"

	describesvc "iq-home/backend/internal/service/describe"
	"iq-home/backend/pkg/respond"
)

type service interface {
	DescribeAll(ctx context.Context) (*describesvc.Result, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/describe
// Starts description generation in the background and returns 202 immediately.
func (h *Handler) DescribeAll(w http.ResponseWriter, r *http.Request) {
	go func() {
		_, _ = h.svc.DescribeAll(context.Background())
	}()
	respond.JSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
