package userhandler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/user"
	"iq-home/backend/internal/middleware"
	"iq-home/backend/pkg/respond"
	"iq-home/backend/pkg/validate"
)

type service interface {
	GetCart(ctx context.Context, userID string) ([]user.CartItem, error)
	AddToCart(ctx context.Context, userID string, productID int64, quantity int) error
	RemoveFromCart(ctx context.Context, userID string, productID int64) error

	GetFavorites(ctx context.Context, userID string) ([]user.FavoriteItem, error)
	ToggleFavorite(ctx context.Context, userID string, productID int64) (string, error)
	RemoveFavorite(ctx context.Context, userID string, productID int64) error

	GetHistory(ctx context.Context, userID string) ([]user.HistoryItem, error)
	AddHistory(ctx context.Context, userID string, productID int64) error
	GetRecommendations(ctx context.Context, userID string) ([]user.RecommendedItem, error)

	GetOrders(ctx context.Context, userID string) ([]user.Order, error)
	GetOrderDetail(ctx context.Context, userID string, orderID int64) (*user.OrderDetail, error)
	Checkout(ctx context.Context, userID string, req user.CheckoutRequest) (*user.CheckoutResult, error)

	GetProfile(ctx context.Context, userID string) (*user.Profile, error)

	UpdateAvatar(ctx context.Context, userID string, update user.AvatarUpdate) error
	DeleteAvatar(ctx context.Context, userID string) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// ─── Cart ────────────────────────────────────────────────────────────────────

// GET /api/user/cart
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	items, err := h.svc.GetCart(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orEmpty(items)})
}

// POST /api/user/cart
func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	var body struct {
		ProductID int64 `json:"productId"`
		Quantity  int   `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.AddToCart(r.Context(), u.ID, body.ProductID, body.Quantity); err != nil {
		respond.BadRequest(w, err.Error())
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// PUT /api/user/cart/{productId}
func (h *Handler) UpdateCartQuantity(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	productID, err := pathInt64(r, "productId")
	if err != nil {
		respond.BadRequest(w, "invalid product id")
		return
	}
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.AddToCart(r.Context(), u.ID, productID, body.Quantity); err != nil {
		respond.BadRequest(w, err.Error())
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// DELETE /api/user/cart/{productId}
func (h *Handler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	productID, err := pathInt64(r, "productId")
	if err != nil {
		respond.BadRequest(w, "invalid product id")
		return
	}
	if err := h.svc.RemoveFromCart(r.Context(), u.ID, productID); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── Favorites ───────────────────────────────────────────────────────────────

// GET /api/user/favorites
func (h *Handler) GetFavorites(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	items, err := h.svc.GetFavorites(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orEmpty(items)})
}

// POST /api/user/favorites
func (h *Handler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	var body struct {
		ProductID      int64 `json:"productId"`
		ProductIDSnake int64 `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if body.ProductID == 0 {
		body.ProductID = body.ProductIDSnake
	}
	if body.ProductID <= 0 {
		respond.BadRequest(w, "productId is required")
		return
	}
	action, err := h.svc.ToggleFavorite(r.Context(), u.ID, body.ProductID)
	if err != nil {
		slog.Error("toggle favorite failed", "user_id", u.ID, "product_id", body.ProductID, "err", err)
		respond.InternalError(w)
		return
	}
	respond.OK(w, user.ToggleResult{Success: true, Action: action})
}

// DELETE /api/user/favorites/{productId}
func (h *Handler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	productID, err := pathInt64(r, "productId")
	if err != nil {
		respond.BadRequest(w, "invalid product id")
		return
	}
	if err := h.svc.RemoveFavorite(r.Context(), u.ID, productID); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── History ─────────────────────────────────────────────────────────────────

// GET /api/user/history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	items, err := h.svc.GetHistory(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orEmpty(items)})
}

// POST /api/user/history
func (h *Handler) AddHistory(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	var body struct {
		ProductID      int64 `json:"product_id"`
		ProductIDCamel int64 `json:"productId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if body.ProductID == 0 {
		body.ProductID = body.ProductIDCamel
	}
	if err := h.svc.AddHistory(r.Context(), u.ID, body.ProductID); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// GET /api/user/recommendations
func (h *Handler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	items, err := h.svc.GetRecommendations(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orEmpty(items)})
}

// ─── Orders ──────────────────────────────────────────────────────────────────

// GET /api/user/orders
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	orders, err := h.svc.GetOrders(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orEmpty(orders)})
}

// GET /api/user/orders/{id}
func (h *Handler) GetOrderDetail(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	orderID, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid order id")
		return
	}
	detail, err := h.svc.GetOrderDetail(r.Context(), u.ID, orderID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	if detail == nil {
		respond.NotFound(w)
		return
	}
	respond.OK(w, detail)
}

// POST /api/user/checkout
func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	var req user.CheckoutRequest
	if !validate.DecodeAndValidate(w, r, &req) {
		return
	}
	result, err := h.svc.Checkout(r.Context(), u.ID, req)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, result)
}

// ─── Session ─────────────────────────────────────────────────────────────────

// GET /api/user/session
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	profile, err := h.svc.GetProfile(r.Context(), u.ID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	// Fill email from the validated JWT (middleware.SupabaseUser) if profile row is missing it.
	if profile.Email == "" && u.Email != "" {
		profile.Email = u.Email
	}
	respond.OK(w, profile)
}

// ─── Avatar ──────────────────────────────────────────────────────────────────

// POST /api/user/avatar
func (h *Handler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	var update user.AvatarUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.UpdateAvatar(r.Context(), u.ID, update); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// DELETE /api/user/avatar
func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	u := mustUser(w, r)
	if u.ID == "" {
		return
	}
	if err := h.svc.DeleteAvatar(r.Context(), u.ID); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// mustUser extracts the authenticated user from context.
// Writes 401 and returns zero value if not present.
func mustUser(w http.ResponseWriter, r *http.Request) middleware.SupabaseUser {
	u, ok := middleware.UserFromContext(r.Context())
	if !ok {
		respond.Unauthorized(w)
		return middleware.SupabaseUser{}
	}
	return u
}

func pathInt64(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

// orEmpty converts a nil slice to an empty slice so JSON encodes [] instead of null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
