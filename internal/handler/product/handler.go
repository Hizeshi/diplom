package producthandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/product"
	"iq-home/backend/pkg/respond"
)

type service interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
	Search(ctx context.Context, params product.SearchParams) (*product.SearchResult, error)
	GetFilters(ctx context.Context) (*product.Filters, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/filters
func (h *Handler) Filters(w http.ResponseWriter, r *http.Request) {
	filters, err := h.svc.GetFilters(r.Context())
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

	p, err := h.svc.GetByID(r.Context(), id)
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
	}

	result, err := h.svc.Search(r.Context(), params)
	if err != nil {
		respond.InternalError(w)
		return
	}

	respond.OK(w, result)
}

// ─── response shapes ─────────────────────────────────────────────────────────

type productResponse struct {
	ID               int64            `json:"id"`
	Article          string           `json:"article"`
	Name             string           `json:"name"`
	Type             string           `json:"product_type"`
	Price            float64          `json:"price"`
	Description      string           `json:"description"`
	Stock            int              `json:"stock"`
	ConfiguratorType string           `json:"configurator_type"`
	Brand            *refResponse     `json:"brand"`
	Color            *refResponse     `json:"color"`
	Series           *refResponse     `json:"series"`
	Images           []product.Image  `json:"images"`
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
