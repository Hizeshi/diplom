package adminhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"iq-home/backend/internal/domain/admin"
	"iq-home/backend/pkg/respond"
)

type service interface {
	ListChats(ctx context.Context) ([]admin.ChatSession, error)
	GetChatHistory(ctx context.Context, sessionID string) ([]admin.ChatMessage, error)
	ToggleHumanMode(ctx context.Context, sessionID string, enabled bool) error
	SendManagerMessage(ctx context.Context, sessionID, text string) error

	ListProducts(ctx context.Context, search string, limit, page int) (*admin.ProductList, error)
	CreateProduct(ctx context.Context, data admin.ProductCreate) (int64, error)
	UpdateProduct(ctx context.Context, id int64, data admin.ProductUpdate) error
	DeleteProduct(ctx context.Context, id int64) error
	ScanDuplicates(ctx context.Context) ([]admin.DuplicatePair, error)

	ListUsers(ctx context.Context, search string, limit, page int) (*admin.UserList, error)
	GetUserDetail(ctx context.Context, id string) (*admin.UserDetail, error)
	UpdateUser(ctx context.Context, id string, data admin.UserUpdate) error
	UpdateUserRole(ctx context.Context, id, role string) error
	UpdateUserEmail(ctx context.Context, id, email string) error
	DeleteUser(ctx context.Context, id string) error

	ListOrders(ctx context.Context) ([]admin.Order, error)
	GetOrderDetail(ctx context.Context, id int64) (*admin.OrderDetail, error)

	ListKnowledge(ctx context.Context) ([]admin.KnowledgeEntry, error)
	UpsertKnowledge(ctx context.Context, req admin.KnowledgeUpsert) error
	DeleteKnowledge(ctx context.Context, id int64) error

	ListContacts(ctx context.Context) ([]admin.Contact, error)
	GetMetadata(ctx context.Context) (*admin.Metadata, error)

	ClearUserCart(ctx context.Context, userID string) error
	ClearUserHistory(ctx context.Context, userID string) error
	UpdateConfiguratorType(ctx context.Context, productID int64, configuratorType string) error
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// ─── Chats ───────────────────────────────────────────────────────────────────

// GET /api/admin/chats
func (h *Handler) ListChats(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.svc.ListChats(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": sessions})
}

// GET /api/admin/chats/{sessionId}
func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	msgs, err := h.svc.GetChatHistory(r.Context(), sessionID)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": msgs})
}

// POST /api/admin/chats/{sessionId}/toggle
func (h *Handler) ToggleHumanMode(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	var body struct {
		IsHumanMode bool `json:"is_human_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.ToggleHumanMode(r.Context(), sessionID, body.IsHumanMode); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// POST /api/admin/chats/{sessionId}/message
func (h *Handler) SendManagerMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if body.Text == "" {
		respond.BadRequest(w, "text is required")
		return
	}
	if err := h.svc.SendManagerMessage(r.Context(), sessionID, body.Text); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// ─── Products ────────────────────────────────────────────────────────────────

// GET /api/admin/products?search=&limit=&page=
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit := queryInt(r, "limit", 20)
	page := queryInt(r, "page", 1)

	result, err := h.svc.ListProducts(r.Context(), search, limit, page)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, result)
}

// POST /api/admin/products
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var data admin.ProductCreate
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	id, err := h.svc.CreateProduct(r.Context(), data)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.Created(w, map[string]int64{"id": id})
}

// PUT /api/admin/products/{id}
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var data admin.ProductUpdate
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.UpdateProduct(r.Context(), id, data); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// DELETE /api/admin/products/{id}
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// POST /api/admin/products/scan-duplicates
func (h *Handler) ScanDuplicates(w http.ResponseWriter, r *http.Request) {
	pairs, err := h.svc.ScanDuplicates(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": pairs})
}

// ─── Users ───────────────────────────────────────────────────────────────────

// GET /api/admin/users?search=&limit=&page=
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit := queryInt(r, "limit", 20)
	page := queryInt(r, "page", 1)

	result, err := h.svc.ListUsers(r.Context(), search, limit, page)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, result)
}

// GET /api/admin/users/{id}
func (h *Handler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.GetUserDetail(r.Context(), id)
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

// PUT /api/admin/users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		admin.UserUpdate
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.UpdateUser(r.Context(), id, body.UserUpdate); err != nil {
		respond.InternalError(w)
		return
	}
	if body.Email != "" {
		_ = h.svc.UpdateUserEmail(r.Context(), id, body.Email)
	}
	if body.Role != "" {
		_ = h.svc.UpdateUserRole(r.Context(), id, body.Role)
	}
	respond.OK(w, map[string]bool{"success": true})
}

// DELETE /api/admin/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteUser(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

// GET /api/admin/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.ListOrders(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": orders})
}

// GET /api/admin/orders/{id}
func (h *Handler) GetOrderDetail(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	detail, err := h.svc.GetOrderDetail(r.Context(), id)
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

// ─── Knowledge ───────────────────────────────────────────────────────────────

// GET /api/admin/knowledge
func (h *Handler) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.ListKnowledge(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": entries})
}

// POST /api/admin/knowledge  (create or update — id in body is optional)
func (h *Handler) UpsertKnowledge(w http.ResponseWriter, r *http.Request) {
	var req admin.KnowledgeUpsert
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if req.Topic == "" || req.Content == "" {
		respond.BadRequest(w, "topic and content are required")
		return
	}
	if err := h.svc.UpsertKnowledge(r.Context(), req); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// DELETE /api/admin/knowledge/{id}
func (h *Handler) DeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	if err := h.svc.DeleteKnowledge(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── Contacts & Metadata ─────────────────────────────────────────────────────

// GET /api/admin/contacts
func (h *Handler) ListContacts(w http.ResponseWriter, r *http.Request) {
	contacts, err := h.svc.ListContacts(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": contacts})
}

// GET /api/admin/metadata
func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	meta, err := h.svc.GetMetadata(r.Context())
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, meta)
}

// ─── User Data Management ─────────────────────────────────────────────────────

// DELETE /api/admin/users/{id}/cart
func (h *Handler) ClearUserCart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.ClearUserCart(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// DELETE /api/admin/users/{id}/history
func (h *Handler) ClearUserHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.ClearUserHistory(r.Context(), id); err != nil {
		respond.InternalError(w)
		return
	}
	respond.NoContent(w)
}

// ─── Products Extra ───────────────────────────────────────────────────────────

// PUT /api/admin/products/{id}/configurator
func (h *Handler) UpdateConfiguratorType(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		respond.BadRequest(w, "invalid id")
		return
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.UpdateConfiguratorType(r.Context(), id, body.Type); err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]bool{"success": true})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func pathInt64(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
