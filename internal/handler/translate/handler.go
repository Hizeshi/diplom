package translatehandler

import (
	"context"
	"net/http"

	translatesvc "iq-home/backend/internal/service/translate"
	"iq-home/backend/pkg/respond"
)

type service interface {
	Translate(ctx context.Context, locale string) (*translatesvc.Result, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/translate?locale=kk
// POST /v1/products/translate?locale=en
// POST /v1/products/translate  (translates both kk and en)
func (h *Handler) Translate(w http.ResponseWriter, r *http.Request) {
	locale := r.URL.Query().Get("locale")

	type localeResult struct {
		Locale string             `json:"locale"`
		Result *translatesvc.Result `json:"result"`
		Error  string             `json:"error,omitempty"`
	}

	var locales []string
	switch locale {
	case "kk", "en":
		locales = []string{locale}
	default:
		locales = []string{"kk", "en"}
	}

	results := make([]localeResult, 0, len(locales))
	for _, loc := range locales {
		res, err := h.svc.Translate(r.Context(), loc)
		lr := localeResult{Locale: loc, Result: res}
		if err != nil {
			lr.Error = err.Error()
		}
		results = append(results, lr)
	}

	respond.OK(w, map[string]any{"results": results})
}
