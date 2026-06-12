package productimagehandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/productimage"
	"iq-home/backend/pkg/respond"
)

const maxUploadSize = 50 << 20 // 50 MB

type service interface {
	Add(ctx context.Context, req productimage.AddRequest) (*productimage.Image, error)
	BulkUpload(ctx context.Context, items []productimage.BulkUploadItem) (*productimage.BulkReport, error)
	UpdateOrder(ctx context.Context, id, displayOrder int) error
	Replace(ctx context.Context, req productimage.ReplaceRequest) (*productimage.Image, error)
	Delete(ctx context.Context, id int) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/images — bulk upload; expects multipart/form-data files.
// Products are matched by the article in each filename ("АРТИКУЛ.jpg",
// "АРТИКУЛ-1.jpg"). Returns {uploaded: [], skipped: [], errors: []}.
func (h *Handler) BulkUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respond.BadRequest(w, "failed to parse form")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		respond.BadRequest(w, "no files provided")
		return
	}

	var items []productimage.BulkUploadItem
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}
		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = "image/jpeg"
		}
		items = append(items, productimage.BulkUploadItem{
			Filename:    fh.Filename,
			Data:        data,
			ContentType: ct,
		})
	}

	report, err := h.svc.BulkUpload(r.Context(), items)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, report)
}

// POST /v1/products/images/item — add single image; multipart: product_id + file
func (h *Handler) ImageAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respond.BadRequest(w, "failed to parse form")
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil || productID <= 0 {
		respond.BadRequest(w, "product_id is required")
		return
	}

	displayOrder, _ := strconv.Atoi(r.FormValue("display_order"))

	fh, _, err := r.FormFile("file")
	if err != nil {
		respond.BadRequest(w, "file is required")
		return
	}
	defer fh.Close()

	header := r.MultipartForm.File["file"][0]
	data, err := io.ReadAll(fh)
	if err != nil {
		respond.InternalError(w)
		return
	}

	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}

	img, err := h.svc.Add(r.Context(), productimage.AddRequest{
		ProductID:    productID,
		Filename:     header.Filename,
		Data:         data,
		ContentType:  ct,
		DisplayOrder: displayOrder,
	})
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.Created(w, img)
}

// PUT /v1/products/images/item — two modes:
//   - multipart/form-data: image_id + file [+ display_order] — replace the photo
//   - JSON {id, display_order} — change display order only
func (h *Handler) ImageUpdate(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		h.imageReplace(w, r)
		return
	}

	var req productimage.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if req.ID == 0 {
		respond.BadRequest(w, "id is required")
		return
	}
	if err := h.svc.UpdateOrder(r.Context(), req.ID, req.DisplayOrder); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

func (h *Handler) imageReplace(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respond.BadRequest(w, "failed to parse form")
		return
	}

	id, err := strconv.Atoi(r.FormValue("image_id"))
	if err != nil || id <= 0 {
		id, err = strconv.Atoi(r.FormValue("id"))
		if err != nil || id <= 0 {
			respond.BadRequest(w, "image_id is required")
			return
		}
	}

	var displayOrder *int
	if v := r.FormValue("display_order"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			displayOrder = &n
		}
	}

	fh, _, err := r.FormFile("file")
	if err != nil {
		// No file — multipart order-only update.
		if displayOrder == nil {
			respond.BadRequest(w, "file or display_order is required")
			return
		}
		if err := h.svc.UpdateOrder(r.Context(), id, *displayOrder); err != nil {
			respond.InternalError(w)
			return
		}
		respond.OK(w, map[string]bool{"success": true})
		return
	}
	defer fh.Close()

	header := r.MultipartForm.File["file"][0]
	data, err := io.ReadAll(fh)
	if err != nil {
		respond.InternalError(w)
		return
	}

	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}

	img, err := h.svc.Replace(r.Context(), productimage.ReplaceRequest{
		ID:           id,
		Filename:     header.Filename,
		Data:         data,
		ContentType:  ct,
		DisplayOrder: displayOrder,
	})
	if errors.Is(err, productimage.ErrNotFound) {
		respond.NotFound(w)
		return
	}
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, img)
}

// DELETE /v1/products/images/item?id=123 (image_id= also accepted)
func (h *Handler) ImageDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		id, err = strconv.Atoi(r.URL.Query().Get("id"))
	}
	if err != nil || id <= 0 {
		id, err = strconv.Atoi(r.URL.Query().Get("image_id"))
	}
	if err != nil || id <= 0 {
		respond.BadRequest(w, "id is required")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, productimage.ErrNotFound) {
			respond.NotFound(w)
			return
		}
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}
