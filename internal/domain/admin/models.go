package admin

import "time"

// ─── Chats ───────────────────────────────────────────────────────────────────

type ChatSession struct {
	SessionID   string `json:"session_id"`
	IsHumanMode bool   `json:"is_human_mode"`
	LastActive  string `json:"last_active"` // formatted "DD.MM HH:MM"
	LastMessage string `json:"last_message"`
}

type ChatMessage struct {
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	SenderType string    `json:"sender_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// ─── Products ────────────────────────────────────────────────────────────────

type Product struct {
	ID               int64      `json:"id"`
	Article          string     `json:"article"`
	Name             string     `json:"name"`
	Type             string     `json:"product_type"`
	Price            float64    `json:"price"`
	Stock            int        `json:"stock"`
	Description      string     `json:"description"`
	BrandID          *int64     `json:"brand_id"`
	BrandName        string     `json:"brand_name"`
	SeriesID         *int64     `json:"series_id"`
	SeriesName       string     `json:"series_name"`
	ColorID          *int64     `json:"color_id"`
	ColorName        string     `json:"color_name"`
	ConfiguratorType string     `json:"configurator_type"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
}

type ProductList struct {
	Items []Product `json:"items"`
	Total int64     `json:"total"`
}

type ProductCreate struct {
	Name    string  `json:"name"`
	Article string  `json:"article"`
	Price   float64 `json:"price"`
	Stock   int     `json:"stock"`
	Type    string  `json:"type"`
}

type ProductUpdate struct {
	Name             string  `json:"name"`
	Article          string  `json:"article"`
	Price            float64 `json:"price"`
	Stock            int     `json:"stock"`
	Description      string  `json:"description"`
	Type             string  `json:"type"`
	BrandID          *int64  `json:"brandId"`
	SeriesID         *int64  `json:"seriesId"`
	ColorID          *int64  `json:"colorId"`
	ConfiguratorType string  `json:"configuratorType"`
}

type DuplicatePair struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	DuplicateID   int64   `json:"duplicate_id"`
	DuplicateName string  `json:"duplicate_name"`
	Similarity    float64 `json:"similarity"`
}

// ─── Users ───────────────────────────────────────────────────────────────────

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

type UserList struct {
	Items []User `json:"items"`
	Total int64  `json:"total"`
}

type UserUpdate struct {
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
}

type UserDetail struct {
	User
	Orders  []UserOrder       `json:"orders"`
	Cart    []UserCartItem    `json:"cart"`
	History []UserHistoryItem `json:"history"`
}

type UserOrder struct {
	ID          int64     `json:"id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserCartItem struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

type UserHistoryItem struct {
	ProductID int64     `json:"product_id"`
	Name      string    `json:"name"`
	ViewedAt  time.Time `json:"viewed_at"`
}

// ─── Orders ──────────────────────────────────────────────────────────────────

type Order struct {
	ID            int64     `json:"id"`
	UserID        *string   `json:"user_id"`
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

// ─── Knowledge ───────────────────────────────────────────────────────────────

type KnowledgeEntry struct {
	ID        int64     `json:"id"`
	Topic     string    `json:"topic"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type KnowledgeUpsert struct {
	ID      *int64 `json:"id"`
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

// ─── Contacts ────────────────────────────────────────────────────────────────

type Contact struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Metadata ────────────────────────────────────────────────────────────────

type Metadata struct {
	ProductsTotal  int64   `json:"products_total"`
	UsersTotal     int64   `json:"users_total"`
	OrdersTotal    int64   `json:"orders_total"`
	OrdersRevenue  float64 `json:"orders_revenue"`
	KnowledgeCount int64   `json:"knowledge_count"`
}

type Option struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
