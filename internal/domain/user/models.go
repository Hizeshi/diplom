package user

import (
	"errors"
	"time"
)

var ErrProductNotFound = errors.New("product not found")

// ─── Cart ────────────────────────────────────────────────────────────────────

type CartItem struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"image_url"`
}

// ─── Favorites ───────────────────────────────────────────────────────────────

type FavoriteItem struct {
	ID        int64        `json:"id"`         // same as product_id — for frontend compatibility
	ProductID int64        `json:"product_id"`
	Article   string       `json:"article"`
	Name      string       `json:"name"`
	Price     float64      `json:"price"`
	ImageURL  string       `json:"image_url"`
	Brand     *NameRef     `json:"brand,omitempty"`
	Color     *NameRef     `json:"color,omitempty"`
	Series    *NameRef     `json:"series,omitempty"`
}

// NameRef is a {name} object used for brand/color/series fields.
type NameRef struct {
	Name string `json:"name"`
}

// ToggleResult tells the caller whether the item was added or removed.
type ToggleResult struct {
	Success bool   `json:"success"`
	Action  string `json:"action"` // "added" | "removed"
}

// ─── History ─────────────────────────────────────────────────────────────────

type HistoryItem struct {
	ProductID int64     `json:"product_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	ImageURL  string    `json:"image_url"`
	ViewedAt  time.Time `json:"viewed_at"`
	Brand     *NameRef  `json:"brand,omitempty"`
	Color     *NameRef  `json:"color,omitempty"`
	Series    *NameRef  `json:"series,omitempty"`
}

type RecommendedItem struct {
	ID       int64    `json:"id"`
	Article  string   `json:"article"`
	Name     string   `json:"name"`
	Price    float64  `json:"price"`
	ImageURL string   `json:"image_url"`
	Brand    *NameRef `json:"brand,omitempty"`
	Color    *NameRef `json:"color,omitempty"`
	Series   *NameRef `json:"series,omitempty"`
}

// ─── Orders ──────────────────────────────────────────────────────────────────

type Order struct {
	ID            int64     `json:"id"`
	Status        string    `json:"status"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	PaymentStatus string    `json:"payment_status"`
	FullName      string    `json:"full_name"`
	Phone         string    `json:"phone"`
	Address       string    `json:"address"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderDetail struct {
	Order
	Items []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID       int64   `json:"product_id"`
	ProductName     string  `json:"product_name"`
	PriceAtPurchase float64 `json:"price"`
	Quantity        int     `json:"quantity"`
}

// ─── Checkout ────────────────────────────────────────────────────────────────

type CheckoutRequest struct {
	Address       string `json:"address"       validate:"required,min=5,max=300"`
	Phone         string `json:"phone"         validate:"required,min=7,max=20"`
	Name          string `json:"full_name"     validate:"required,min=2,max=100"`
	PaymentMethod string `json:"paymentMethod" validate:"required,oneof=card cash kaspi other"`
}

type CheckoutResult struct {
	Success    bool   `json:"success"`
	OrderID    int64  `json:"orderId"`
	PaymentURL string `json:"paymentUrl,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ─── Session ─────────────────────────────────────────────────────────────────

type Session struct {
	SessionID   string `json:"session_id"`
	IsHumanMode bool   `json:"is_human_mode"`
}

// ─── Profile ─────────────────────────────────────────────────────────────────

type Profile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}

// ─── Avatar ──────────────────────────────────────────────────────────────────

type AvatarUpdate struct {
	URL  string `json:"avatar_url"`
	Path string `json:"avatar_path"`
}
