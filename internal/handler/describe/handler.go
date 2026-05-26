package describehandler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	describesvc "iq-home/backend/internal/service/describe"
	"iq-home/backend/pkg/respond"
)

type service interface {
	DescribeAll(ctx context.Context) (*describesvc.Result, error)
}

type Handler struct {
	svc service
	log *slog.Logger
}

func New(svc service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// POST /v1/products/describe
// Starts description generation in the background and returns 202 immediately.
func (h *Handler) DescribeAll(w http.ResponseWriter, r *http.Request) {
	go func() {
		start := time.Now()
		h.log.Info("describe: job started")
		result, err := h.svc.DescribeAll(context.Background())
		if err != nil {
			h.log.Error("describe: job failed", "err", err, "duration", time.Since(start).String())
			return
		}
		h.log.Info("describe: job finished",
			"total", result.Total,
			"success", result.Success,
			"failed", result.Failed,
			"duration", time.Since(start).String(),
		)
	}()
	respond.JSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}
