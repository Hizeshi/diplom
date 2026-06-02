package usersvc

import (
	"context"
	"fmt"

	"iq-home/backend/internal/domain/user"
)

type repo interface {
	GetCart(ctx context.Context, userID string) ([]user.CartItem, error)
	UpsertCartItem(ctx context.Context, userID string, productID int64, quantity int) error
	RemoveCartItem(ctx context.Context, userID string, productID int64) error

	GetFavorites(ctx context.Context, userID string) ([]user.FavoriteItem, error)
	ToggleFavorite(ctx context.Context, userID string, productID int64) (string, error)
	RemoveFavorite(ctx context.Context, userID string, productID int64) error

	GetHistory(ctx context.Context, userID string) ([]user.HistoryItem, error)
	AddHistory(ctx context.Context, userID string, productID int64) error
	GetRecommendations(ctx context.Context, userID, locale string) ([]user.RecommendedItem, error)

	GetOrders(ctx context.Context, userID string) ([]user.Order, error)
	GetOrderDetail(ctx context.Context, userID string, orderID int64) (*user.OrderDetail, error)
	Checkout(ctx context.Context, userID string, req user.CheckoutRequest) (int64, float64, error)

	GetProfile(ctx context.Context, userID string) (*user.Profile, error)
	GetSession(ctx context.Context, userID string) (*user.Session, error)

	GetAvatarPath(ctx context.Context, userID string) (string, error)
	UpdateAvatar(ctx context.Context, userID, url, path string) error
	ClearAvatar(ctx context.Context, userID string) error
}

// storage is satisfied by client/supabase.Client.
type storage interface {
	Delete(ctx context.Context, bucket string, paths []string) error
}

type Service struct {
	repo           repo
	storage        storage
	paymentBaseURL string
}

func New(repo repo, storage storage, paymentBaseURL string) *Service {
	return &Service{repo: repo, storage: storage, paymentBaseURL: paymentBaseURL}
}

// ─── Cart ────────────────────────────────────────────────────────────────────

func (s *Service) GetCart(ctx context.Context, userID string) ([]user.CartItem, error) {
	return s.repo.GetCart(ctx, userID)
}

func (s *Service) AddToCart(ctx context.Context, userID string, productID int64, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	return s.repo.UpsertCartItem(ctx, userID, productID, quantity)
}

func (s *Service) RemoveFromCart(ctx context.Context, userID string, productID int64) error {
	return s.repo.RemoveCartItem(ctx, userID, productID)
}

// ─── Favorites ───────────────────────────────────────────────────────────────

func (s *Service) GetFavorites(ctx context.Context, userID string) ([]user.FavoriteItem, error) {
	return s.repo.GetFavorites(ctx, userID)
}

func (s *Service) ToggleFavorite(ctx context.Context, userID string, productID int64) (string, error) {
	return s.repo.ToggleFavorite(ctx, userID, productID)
}

func (s *Service) RemoveFavorite(ctx context.Context, userID string, productID int64) error {
	return s.repo.RemoveFavorite(ctx, userID, productID)
}

// ─── History ─────────────────────────────────────────────────────────────────

func (s *Service) GetHistory(ctx context.Context, userID string) ([]user.HistoryItem, error) {
	return s.repo.GetHistory(ctx, userID)
}

func (s *Service) AddHistory(ctx context.Context, userID string, productID int64) error {
	return s.repo.AddHistory(ctx, userID, productID)
}

func (s *Service) GetRecommendations(ctx context.Context, userID, locale string) ([]user.RecommendedItem, error) {
	return s.repo.GetRecommendations(ctx, userID, locale)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (s *Service) GetOrders(ctx context.Context, userID string) ([]user.Order, error) {
	return s.repo.GetOrders(ctx, userID)
}

func (s *Service) GetOrderDetail(ctx context.Context, userID string, orderID int64) (*user.OrderDetail, error) {
	return s.repo.GetOrderDetail(ctx, userID, orderID)
}

func (s *Service) Checkout(ctx context.Context, userID string, req user.CheckoutRequest) (*user.CheckoutResult, error) {
	orderID, total, err := s.repo.Checkout(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("user service: checkout: %w", err)
	}

	result := &user.CheckoutResult{Success: true, OrderID: orderID}

	needsPaymentURL := (req.PaymentMethod == "card" || req.PaymentMethod == "kaspi") && s.paymentBaseURL != ""
	if needsPaymentURL {
		result.PaymentURL = fmt.Sprintf("%s/?orderId=%d&amount=%d", s.paymentBaseURL, orderID, int64(total))
	} else {
		result.Message = "Заказ принят. Менеджер свяжется с вами."
	}

	return result, nil
}

// ─── Profile / Session ───────────────────────────────────────────────────────

func (s *Service) GetProfile(ctx context.Context, userID string) (*user.Profile, error) {
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) GetSession(ctx context.Context, userID string) (*user.Session, error) {
	return s.repo.GetSession(ctx, userID)
}

// ─── Avatar ──────────────────────────────────────────────────────────────────

func (s *Service) UpdateAvatar(ctx context.Context, userID string, update user.AvatarUpdate) error {
	return s.repo.UpdateAvatar(ctx, userID, update.URL, update.Path)
}

func (s *Service) DeleteAvatar(ctx context.Context, userID string) error {
	path, err := s.repo.GetAvatarPath(ctx, userID)
	if err != nil {
		return fmt.Errorf("user service: delete avatar: %w", err)
	}

	if path != "" {
		if err := s.storage.Delete(ctx, "avatars", []string{path}); err != nil {
			// Non-fatal: log would go here; proceed to clear DB record.
			_ = err
		}
	}

	return s.repo.ClearAvatar(ctx, userID)
}
