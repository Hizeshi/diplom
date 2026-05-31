package producthandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/product"
	"iq-home/backend/internal/middleware"
	"iq-home/backend/pkg/respond"
	"iq-home/backend/pkg/validate"
)

type service interface {
	GetByID(ctx context.Context, id int64, locale string) (*product.Product, error)
	Search(ctx context.Context, params product.SearchParams) (*product.SearchResult, error)
	GetFilters(ctx context.Context, locale string) (*product.Filters, error)
	UpdateProductI18n(ctx context.Context, id int64, upd product.I18nUpdate) error
	UpdateSeriesI18n(ctx context.Context, id int64, upd product.I18nUpdate) error
	UpdateBrandI18n(ctx context.Context, id int64, upd product.I18nUpdate) error
	UpdateColorI18n(ctx context.Context, id int64, upd product.I18nUpdate) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/filters
func (h *Handler) Filters(w http.ResponseWriter, r *http.Request) {
	locale := middleware.LocaleFromContext(r.Context())
	filters, err := h.svc.GetFilters(r.Context(), locale)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, filters)
}

// GET /api/products/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.BadRequest(w, "invalid product id")
		return
	}
	locale := middleware.LocaleFromContext(r.Context())
	p, err := h.svc.GetByID(r.Context(), id, locale)
	if err != nil {
		respond.InternalError(w)
		return
	}
	if p == nil {
		respond.NotFound(w)
		return
	}
	respond.OK(w, toProductResponse(p))
}

// POST /api/products/search
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Search   string   `json:"search"`
		Limit    int      `json:"limit"`
		Page     int      `json:"page"`
		MinPrice *float64 `json:"minPrice"`
		MaxPrice *float64 `json:"maxPrice"`
		BrandID  *int64   `json:"brandId"`
		ColorID  *int64   `json:"colorId"`
		Type     *string  `json:"type"`
		SeriesID *int64   `json:"seriesId"`
		SortBy   string   `json:"sortBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid request body")
		return
	}

	locale := middleware.LocaleFromContext(r.Context())
	params := product.SearchParams{
		Query:    body.Search,
		Limit:    body.Limit,
		Page:     body.Page,
		MinPrice: body.MinPrice,
		MaxPrice: body.MaxPrice,
		BrandID:  body.BrandID,
		ColorID:  body.ColorID,
		Type:     body.Type,
		SeriesID: body.SeriesID,
		SortBy:   body.SortBy,
		Locale:   locale,
	}

	result, err := h.svc.Search(r.Context(), params)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, result)
}

// ─── Admin i18n endpoints ────────────────────────────────────────────────────

// PUT /api/admin/products/{id}/i18n
func (h *Handler) UpdateProductI18n(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var upd product.I18nUpdate
	if !validate.DecodeAndValidate(w, r, &upd) {
		return
	}
	if err := h.svc.UpdateProductI18n(r.Context(), id, upd); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// PUT /api/admin/series/{id}/i18n
func (h *Handler) UpdateSeriesI18n(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var upd product.I18nUpdate
	if !validate.DecodeAndValidate(w, r, &upd) {
		return
	}
	if err := h.svc.UpdateSeriesI18n(r.Context(), id, upd); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// PUT /api/admin/brands/{id}/i18n
func (h *Handler) UpdateBrandI18n(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var upd product.I18nUpdate
	if !validate.DecodeAndValidate(w, r, &upd) {
		return
	}
	if err := h.svc.UpdateBrandI18n(r.Context(), id, upd); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// PUT /api/admin/colors/{id}/i18n
func (h *Handler) UpdateColorI18n(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var upd product.I18nUpdate
	if !validate.DecodeAndValidate(w, r, &upd) {
		return
	}
	if err := h.svc.UpdateColorI18n(r.Context(), id, upd); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// ─── response shapes ─────────────────────────────────────────────────────────

type productResponse struct {
	ID               int64           `json:"id"`
	Article          string          `json:"article"`
	Name             string          `json:"name"`
	Type             string          `json:"product_type"`
	Price            float64         `json:"price"`
	Description      string          `json:"description"`
	Stock            int             `json:"stock"`
	ConfiguratorType string          `json:"configurator_type"`
	Brand            *refResponse    `json:"brand"`
	Color            *refResponse    `json:"color"`
	Series           *refResponse    `json:"series"`
	Images           []product.Image `json:"images"`
	ModelURL         string          `json:"model_url,omitempty"`
}

type refResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func toProductResponse(p *product.Product) productResponse {
	resp := productResponse{
		ID:               p.ID,
		Article:          p.Article,
		Name:             p.Name,
		Type:             p.Type,
		Price:            p.Price,
		Description:      p.Description,
		Stock:            p.Stock,
		ConfiguratorType: p.ConfiguratorType,
		Images:           p.Images,
		ModelURL:         p.ModelURL,
	}
	if p.Brand != nil {
		resp.Brand = &refResponse{ID: p.Brand.ID, Name: p.Brand.Name}
	}
	if p.Color != nil {
		resp.Color = &refResponse{ID: p.Color.ID, Name: p.Color.Name}
	}
	if p.Series != nil {
		resp.Series = &refResponse{ID: p.Series.ID, Name: p.Series.Name}
	}
	return resp
}
