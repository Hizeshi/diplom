package health

import (
	"net/http"

	"iq-home/backend/pkg/respond"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, map[string]string{"status": "ok"})
}
