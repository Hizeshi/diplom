package vectorizehandler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	vectorizesvc "iq-home/backend/internal/service/vectorize"
	"iq-home/backend/pkg/respond"
)

type service interface {
	VectorizeAll(ctx context.Context) (*vectorizesvc.Result, error)
}

type Handler struct {
	svc service
	log *slog.Logger
}

func New(svc service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// POST /v1/products/vectorize
// Starts vectorization in the background and returns 202 immediately.
func (h *Handler) VectorizeAll(w http.ResponseWriter, r *http.Request) {
	go func() {
		start := time.Now()
		h.log.Info("vectorize: job started")
		result, err := h.svc.VectorizeAll(context.Background())
		if err != nil {
			h.log.Error("vectorize: job failed", "err", err, "duration", time.Since(start).String())
			return
		}
		h.log.Info("vectorize: job finished",
			"total", result.Total,
			"success", result.Success,
			"failed", result.Failed,
			"duration", time.Since(start).String(),
		)
	}()
	respond.JSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
