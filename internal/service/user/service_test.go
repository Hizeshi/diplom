package usersvc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"iq-home/backend/internal/domain/user"
	usersvc "iq-home/backend/internal/service/user"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	cart        []user.CartItem
	orderID     int64
	avatarPath  string
	err         error
	upsertCalls int
	clearCalled bool
}

func (m *mockRepo) GetCart(_ context.Context, _ string) ([]user.CartItem, error) {
	return m.cart, m.err
}
func (m *mockRepo) UpsertCartItem(_ context.Context, _ string, _ int64, _ int) error {
	m.upsertCalls++
	return m.err
}
func (m *mockRepo) RemoveCartItem(_ context.Context, _ string, _ int64) error { return m.err }
func (m *mockRepo) GetFavorites(_ context.Context, _ string) ([]user.FavoriteItem, error) {
	return nil, m.err
}
func (m *mockRepo) ToggleFavorite(_ context.Context, _ string, _ int64) (string, error) {
	return "added", m.err
}
func (m *mockRepo) RemoveFavorite(_ context.Context, _ string, _ int64) error { return m.err }
func (m *mockRepo) GetHistory(_ context.Context, _ string) ([]user.HistoryItem, error) {
	return nil, m.err
}
func (m *mockRepo) AddHistory(_ context.Context, _ string, _ int64) error { return m.err }
func (m *mockRepo) GetRecommendations(_ context.Context, _, _ string) ([]user.RecommendedItem, error) {
	return nil, m.err
}
func (m *mockRepo) GetOrders(_ context.Context, _ string) ([]user.Order, error) {
	return nil, m.err
}
func (m *mockRepo) GetOrderDetail(_ context.Context, _ string, _ int64) (*user.OrderDetail, error) {
	return nil, m.err
}
func (m *mockRepo) Checkout(_ context.Context, _ string, _ user.CheckoutRequest) (int64, float64, error) {
	return m.orderID, 0, m.err
}
func (m *mockRepo) GetProfile(_ context.Context, _ string) (*user.Profile, error) {
	return &user.Profile{}, m.err
}
func (m *mockRepo) GetSession(_ context.Context, _ string) (*user.Session, error) {
	return nil, m.err
}
func (m *mockRepo) GetAvatarPath(_ context.Context, _ string) (string, error) {
	return m.avatarPath, m.err
}
func (m *mockRepo) UpdateAvatar(_ context.Context, _, _, _ string) error { return m.err }
func (m *mockRepo) ClearAvatar(_ context.Context, _ string) error {
	m.clearCalled = true
	return m.err
}

type mockStorage struct {
	deleteCalled bool
	err          error
}

func (m *mockStorage) Delete(_ context.Context, _ string, _ []string) error {
	m.deleteCalled = true
	return m.err
}

func newSvc(repo *mockRepo, storage *mockStorage, paymentURL string) *usersvc.Service {
	return usersvc.New(repo, storage, paymentURL)
}

// ─── AddToCart ────────────────────────────────────────────────────────────────

func TestAddToCart_ValidQuantity(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(repo, &mockStorage{}, "")

	err := svc.AddToCart(context.Background(), "user-1", 42, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upsertCalls != 1 {
		t.Errorf("expected 1 upsert call, got %d", repo.upsertCalls)
	}
}

func TestAddToCart_ZeroQuantity_ReturnsError(t *testing.T) {
	svc := newSvc(&mockRepo{}, &mockStorage{}, "")

	err := svc.AddToCart(context.Background(), "user-1", 42, 0)
	if err == nil {
		t.Fatal("expected error for quantity=0")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got: %s", err.Error())
	}
}

func TestAddToCart_NegativeQuantity_ReturnsError(t *testing.T) {
	svc := newSvc(&mockRepo{}, &mockStorage{}, "")

	err := svc.AddToCart(context.Background(), "user-1", 42, -5)
	if err == nil {
		t.Fatal("expected error for negative quantity")
	}
}

// ─── Checkout ─────────────────────────────────────────────────────────────────

func TestCheckout_CardWithPaymentURL(t *testing.T) {
	svc := newSvc(&mockRepo{orderID: 7}, &mockStorage{}, "https://pay.example.com")

	req := user.CheckoutRequest{
		Name:          "Алибек",
		Phone:         "+77001234567",
		Address:       "Алматы",
		PaymentMethod: "card",
	}
	result, err := svc.Checkout(context.Background(), "user-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.OrderID != 7 {
		t.Errorf("expected orderID=7, got %d", result.OrderID)
	}
	if !strings.Contains(result.PaymentURL, "orderId=7") {
		t.Errorf("expected payment URL with orderId, got: %s", result.PaymentURL)
	}
}

func TestCheckout_CashNoPaymentURL(t *testing.T) {
	svc := newSvc(&mockRepo{orderID: 3}, &mockStorage{}, "")

	req := user.CheckoutRequest{
		Name:          "Алибек",
		Phone:         "+77001234567",
		Address:       "Алматы",
		PaymentMethod: "cash",
	}
	result, err := svc.Checkout(context.Background(), "user-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentURL != "" {
		t.Errorf("expected no payment URL for cash, got: %s", result.PaymentURL)
	}
	if result.Message == "" {
		t.Error("expected message for cash order")
	}
}

func TestCheckout_RepoError(t *testing.T) {
	svc := newSvc(&mockRepo{err: errors.New("cart is empty")}, &mockStorage{}, "")

	_, err := svc.Checkout(context.Background(), "user-1", user.CheckoutRequest{})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// ─── DeleteAvatar ─────────────────────────────────────────────────────────────

func TestDeleteAvatar_WithPath_DeletesFromStorage(t *testing.T) {
	repo := &mockRepo{avatarPath: "avatars/user-1/photo.jpg"}
	storage := &mockStorage{}
	svc := newSvc(repo, storage, "")

	err := svc.DeleteAvatar(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !storage.deleteCalled {
		t.Error("expected storage.Delete to be called")
	}
	if !repo.clearCalled {
		t.Error("expected repo.ClearAvatar to be called")
	}
}

func TestDeleteAvatar_NoPath_SkipsStorage(t *testing.T) {
	repo := &mockRepo{avatarPath: ""}
	storage := &mockStorage{}
	svc := newSvc(repo, storage, "")

	err := svc.DeleteAvatar(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storage.deleteCalled {
		t.Error("expected storage.Delete NOT to be called when path is empty")
	}
	if !repo.clearCalled {
		t.Error("expected repo.ClearAvatar to be called even without path")
	}
}

func TestDeleteAvatar_StorageFails_StillClearsDB(t *testing.T) {
	// Storage failure is non-fatal — DB must still be cleared
	repo := &mockRepo{avatarPath: "some/path.jpg"}
	storage := &mockStorage{err: errors.New("storage unavailable")}
	svc := newSvc(repo, storage, "")

	err := svc.DeleteAvatar(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.clearCalled {
		t.Error("expected DB to be cleared even after storage failure")
	}
}
