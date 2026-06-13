package adminsvc

import (
	"context"
	"fmt"

	"iq-home/backend/internal/domain/admin"
)

// ─── Dependency interfaces ────────────────────────────────────────────────────

type repo interface {
	ListChats(ctx context.Context) ([]admin.ChatSession, error)
	GetChatHistory(ctx context.Context, sessionID string) ([]admin.ChatMessage, error)
	ToggleHumanMode(ctx context.Context, sessionID string, enabled bool) error
	SendManagerMessage(ctx context.Context, sessionID, text string) error

	ListProducts(ctx context.Context, search string, limit, page int) (*admin.ProductList, error)
	GetProduct(ctx context.Context, id int64) (*admin.Product, error)
	CreateProduct(ctx context.Context, data admin.ProductCreate) (int64, error)
	UpdateProduct(ctx context.Context, id int64, data admin.ProductUpdate) error
	DeleteProduct(ctx context.Context, id int64) error
	ScanDuplicates(ctx context.Context) ([]admin.DuplicatePair, error)

	ListUsers(ctx context.Context, search string, limit, page int) (*admin.UserList, error)
	GetUserDetail(ctx context.Context, id string) (*admin.UserDetail, error)
	UpdateUser(ctx context.Context, id string, data admin.UserUpdate) error
	UpdateUserRole(ctx context.Context, id, role string) error
	DeleteUserData(ctx context.Context, id string) error

	ListOrders(ctx context.Context) ([]admin.Order, error)
	GetOrderDetail(ctx context.Context, id int64) (*admin.OrderDetail, error)
	UpdateOrderStatus(ctx context.Context, id int64, status string) error
	DeleteOrder(ctx context.Context, id int64) error

	ListKnowledge(ctx context.Context) ([]admin.KnowledgeEntry, error)
	UpsertKnowledge(ctx context.Context, id *int64, topic, content string, embedding []float32) error
	DeleteKnowledge(ctx context.Context, id int64) error

	ListContacts(ctx context.Context) ([]admin.Contact, error)
	GetMetadata(ctx context.Context) (*admin.Metadata, error)
	GetStats(ctx context.Context) (*admin.Stats, error)

	ClearUserCart(ctx context.Context, userID string) error
	ClearUserHistory(ctx context.Context, userID string) error
	DeleteCartItem(ctx context.Context, userID string, productID int64) error
	DeleteHistoryItem(ctx context.Context, userID string, productID int64) error
	DeleteFavoriteItem(ctx context.Context, userID string, productID int64) error
	ClearUserFavorites(ctx context.Context, userID string) error
	UpdateConfiguratorType(ctx context.Context, productID int64, configuratorType string) error
}

// embedder is satisfied by client/openai.EmbedAdapter.
type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// authProvider deletes a user from Supabase Auth (satisfied by client/supabase.Client).
type authProvider interface {
	AdminDeleteUser(ctx context.Context, userID string) error
	AdminUpdateEmail(ctx context.Context, userID, email string) error
}

// ─── Service ─────────────────────────────────────────────────────────────────

type Service struct {
	repo     repo
	embedder embedder
	auth     authProvider
}

func New(repo repo, embedder embedder, auth authProvider) *Service {
	return &Service{repo: repo, embedder: embedder, auth: auth}
}

// ─── Chats ───────────────────────────────────────────────────────────────────

func (s *Service) ListChats(ctx context.Context) ([]admin.ChatSession, error) {
	return s.repo.ListChats(ctx)
}

func (s *Service) GetChatHistory(ctx context.Context, sessionID string) ([]admin.ChatMessage, error) {
	return s.repo.GetChatHistory(ctx, sessionID)
}

func (s *Service) ToggleHumanMode(ctx context.Context, sessionID string, enabled bool) error {
	return s.repo.ToggleHumanMode(ctx, sessionID, enabled)
}

func (s *Service) SendManagerMessage(ctx context.Context, sessionID, text string) error {
	return s.repo.SendManagerMessage(ctx, sessionID, text)
}

// ─── Products ────────────────────────────────────────────────────────────────

func (s *Service) ListProducts(ctx context.Context, search string, limit, page int) (*admin.ProductList, error) {
	return s.repo.ListProducts(ctx, search, limit, page)
}

func (s *Service) GetProduct(ctx context.Context, id int64) (*admin.Product, error) {
	return s.repo.GetProduct(ctx, id)
}

func (s *Service) CreateProduct(ctx context.Context, data admin.ProductCreate) (int64, error) {
	return s.repo.CreateProduct(ctx, data)
}

func (s *Service) UpdateProduct(ctx context.Context, id int64, data admin.ProductUpdate) error {
	return s.repo.UpdateProduct(ctx, id, data)
}

func (s *Service) DeleteProduct(ctx context.Context, id int64) error {
	return s.repo.DeleteProduct(ctx, id)
}

func (s *Service) ScanDuplicates(ctx context.Context) ([]admin.DuplicatePair, error) {
	return s.repo.ScanDuplicates(ctx)
}

// ─── Users ───────────────────────────────────────────────────────────────────

func (s *Service) ListUsers(ctx context.Context, search string, limit, page int) (*admin.UserList, error) {
	return s.repo.ListUsers(ctx, search, limit, page)
}

func (s *Service) GetUserDetail(ctx context.Context, id string) (*admin.UserDetail, error) {
	return s.repo.GetUserDetail(ctx, id)
}

func (s *Service) UpdateUser(ctx context.Context, id string, data admin.UserUpdate) error {
	return s.repo.UpdateUser(ctx, id, data)
}

func (s *Service) UpdateUserRole(ctx context.Context, id, role string) error {
	return s.repo.UpdateUserRole(ctx, id, role)
}

func (s *Service) UpdateUserEmail(ctx context.Context, id, email string) error {
	return s.auth.AdminUpdateEmail(ctx, id, email)
}

// DeleteUser removes the user from both Supabase Auth and local DB.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if err := s.auth.AdminDeleteUser(ctx, id); err != nil {
		return fmt.Errorf("admin: delete user auth: %w", err)
	}
	return s.repo.DeleteUserData(ctx, id)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (s *Service) ListOrders(ctx context.Context) ([]admin.Order, error) {
	return s.repo.ListOrders(ctx)
}

func (s *Service) GetOrderDetail(ctx context.Context, id int64) (*admin.OrderDetail, error) {
	return s.repo.GetOrderDetail(ctx, id)
}

func (s *Service) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	return s.repo.UpdateOrderStatus(ctx, id, status)
}

func (s *Service) DeleteOrder(ctx context.Context, id int64) error {
	return s.repo.DeleteOrder(ctx, id)
}

// ─── Knowledge ───────────────────────────────────────────────────────────────

func (s *Service) ListKnowledge(ctx context.Context) ([]admin.KnowledgeEntry, error) {
	return s.repo.ListKnowledge(ctx)
}

// UpsertKnowledge generates an embedding then saves the entry.
func (s *Service) UpsertKnowledge(ctx context.Context, req admin.KnowledgeUpsert) error {
	embedding, err := s.embedder.Embed(ctx, req.Topic+"\n"+req.Content)
	if err != nil {
		return fmt.Errorf("admin: knowledge embed: %w", err)
	}
	return s.repo.UpsertKnowledge(ctx, req.ID, req.Topic, req.Content, embedding)
}

func (s *Service) DeleteKnowledge(ctx context.Context, id int64) error {
	return s.repo.DeleteKnowledge(ctx, id)
}

// ─── User Data Management ─────────────────────────────────────────────────────

func (s *Service) ClearUserCart(ctx context.Context, userID string) error {
	return s.repo.ClearUserCart(ctx, userID)
}

func (s *Service) ClearUserHistory(ctx context.Context, userID string) error {
	return s.repo.ClearUserHistory(ctx, userID)
}

func (s *Service) DeleteCartItem(ctx context.Context, userID string, productID int64) error {
	return s.repo.DeleteCartItem(ctx, userID, productID)
}

func (s *Service) DeleteHistoryItem(ctx context.Context, userID string, productID int64) error {
	return s.repo.DeleteHistoryItem(ctx, userID, productID)
}

func (s *Service) DeleteFavoriteItem(ctx context.Context, userID string, productID int64) error {
	return s.repo.DeleteFavoriteItem(ctx, userID, productID)
}

func (s *Service) ClearUserFavorites(ctx context.Context, userID string) error {
	return s.repo.ClearUserFavorites(ctx, userID)
}

// ─── Products Extra ───────────────────────────────────────────────────────────

func (s *Service) UpdateConfiguratorType(ctx context.Context, productID int64, configuratorType string) error {
	return s.repo.UpdateConfiguratorType(ctx, productID, configuratorType)
}

// ─── Contacts ────────────────────────────────────────────────────────────────

func (s *Service) ListContacts(ctx context.Context) ([]admin.Contact, error) {
	return s.repo.ListContacts(ctx)
}

// ─── Metadata ────────────────────────────────────────────────────────────────

func (s *Service) GetMetadata(ctx context.Context) (*admin.Metadata, error) {
	return s.repo.GetMetadata(ctx)
}

// GetStats возвращает детальную статистику для дашборда админки.
func (s *Service) GetStats(ctx context.Context) (*admin.Stats, error) {
	return s.repo.GetStats(ctx)
}
