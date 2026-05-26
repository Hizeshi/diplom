package productimagehandler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/productimage"
	"iq-home/backend/pkg/respond"
)

const maxUploadSize = 50 << 20 // 50 MB

type service interface {
	Add(ctx context.Context, req productimage.AddRequest) (*productimage.Image, error)
	BulkUpload(ctx context.Context, items []productimage.BulkUploadItem) ([]*productimage.Image, error)
	UpdateOrder(ctx context.Context, id, displayOrder int) error
	Delete(ctx context.Context, id int) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// POST /v1/products/images — bulk upload; expects multipart/form-data files
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

	uploaded, err := h.svc.BulkUpload(r.Context(), items)
	if err != nil {
		respond.InternalError(w)
		return
	}
	if uploaded == nil {
		uploaded = []*productimage.Image{}
	}
	respond.OK(w, map[string]any{"uploaded": len(uploaded), "items": uploaded})
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

// PUT /v1/products/images/item — update display_order; JSON body: {id, display_order}
func (h *Handler) ImageUpdate(w http.ResponseWriter, r *http.Request) {
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

// DELETE /v1/products/images/item?id=123
func (h *Handler) ImageDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		id, err = strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil || id <= 0 {
			respond.BadRequest(w, "id is required")
			return
		}
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}
