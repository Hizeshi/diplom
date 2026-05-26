package vectorizehandler

import (
	"context"
	"net/http"

	"iq-home/backend/internal/service/vectorize"
	"iq-home/backend/pkg/respond"
)

type service interface {
	VectorizeAll(ctx context.Context) (*vectorizesvc.Result, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/vectorize
// Starts vectorization in the background and returns 202 immediately.
func (h *Handler) VectorizeAll(w http.ResponseWriter, r *http.Request) {
	go func() {
		// Use a detached context so the job survives request cancellation.
		_, _ = h.svc.VectorizeAll(context.Background())
	}()
	respond.JSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
