package productimporthandler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"iq-home/backend/internal/domain/productimport"
	"iq-home/backend/pkg/respond"
)

const maxImportSize = 20 << 20 // 20 MB

type service interface {
	ImportCSV(ctx context.Context, data []byte) (*productimport.Result, error)
	ImportExcel(ctx context.Context, data []byte) (*productimport.Result, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/import
// Accepts multipart/form-data with field "file" (CSV or XLSX).
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxImportSize); err != nil {
		respond.BadRequest(w, "failed to parse form")
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		respond.BadRequest(w, "file is required")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		respond.InternalError(w)
		return
	}

	filename := strings.ToLower(header.Filename)
	ct := strings.ToLower(header.Header.Get("Content-Type"))

	var result *productimport.Result

	switch {
	case strings.HasSuffix(filename, ".xlsx") ||
		strings.Contains(ct, "spreadsheetml") ||
		strings.Contains(ct, "ms-excel"):
		result, err = h.svc.ImportExcel(r.Context(), data)
	default:
		// CSV — also catches .csv and text/plain
		result, err = h.svc.ImportCSV(r.Context(), data)
	}

	if err != nil {
		respond.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respond.OK(w, result)
}
