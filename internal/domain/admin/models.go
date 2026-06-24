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
	Name             string     `json:"name_raw"`
	Type             string     `json:"product_type"`
	Price            float64    `json:"price"`
	Stock            int        `json:"stock"`
	Description      string     `json:"description"`
	BrandID          *int64     `json:"brand_id"`
	BrandName        string     `json:"brand"`
	SeriesID         *int64     `json:"series_id"`
	SeriesName       string     `json:"series"`
	ColorID          *int64     `json:"color_id"`
	ColorName        string     `json:"color_name"`
	ConfiguratorType string         `json:"configurator_type"`
	IsActive         bool           `json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	Images           []ProductImage `json:"images"`
}

type ProductImage struct {
	ID   int64  `json:"id"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

type ProductList struct {
	Items []Product `json:"items"`
	Total int64     `json:"total"`
}

type ProductCreate struct {
	Name        string  `json:"name"`
	Article     string  `json:"article"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	BrandID     *int64  `json:"brandId"`
	SeriesID    *int64  `json:"seriesId"`
	ColorID     *int64  `json:"colorId"`
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
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	RegisteredAt time.Time `json:"registered_at"`
	OrdersCount  int64     `json:"orders_count"`
	TotalSpent   float64   `json:"total_spent"`
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
	Orders    []UserOrder         `json:"orders"`
	Cart      []UserCartItem      `json:"cart"`
	History   []UserHistoryItem   `json:"history"`
	Favorites []UserFavoriteItem  `json:"favorites"`
}

type UserFavoriteItem struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Article   string  `json:"article"`
	Image     string  `json:"image"`
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

// ─── Stats (детальная статистика для дашборда админки) ──────────────────────

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type DailyOrders struct {
	Date    string  `json:"date"` // YYYY-MM-DD
	Count   int64   `json:"count"`
	Revenue float64 `json:"revenue"`
}

type TopProduct struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int64   `json:"quantity"`
	Revenue   float64 `json:"revenue"`
}

type Stats struct {
	// Каталог
	ProductsTotal    int64 `json:"products_total"`
	ProductsActive   int64 `json:"products_active"`
	ProductsNoImages int64 `json:"products_no_images"`
	ProductsLowStock int64 `json:"products_low_stock"` // stock <= 5

	// Пользователи
	UsersTotal  int64 `json:"users_total"`
	UsersNew30d int64 `json:"users_new_30d"`

	// Заказы
	OrdersTotal    int64         `json:"orders_total"`
	Orders30d      int64         `json:"orders_30d"`
	RevenueTotal   float64       `json:"revenue_total"`
	RevenuePaid    float64       `json:"revenue_paid"`
	Revenue30d     float64       `json:"revenue_30d"`
	AvgOrderValue  float64       `json:"avg_order_value"`
	OrdersByStatus []StatusCount `json:"orders_by_status"`
	PaymentMethods []StatusCount `json:"payment_methods"`
	OrdersDaily    []DailyOrders `json:"orders_daily"` // последние 14 дней
	TopProducts    []TopProduct  `json:"top_products"` // топ-5 по выручке

	// Чат и обратная связь
	ChatSessionsTotal int64 `json:"chat_sessions_total"`
	ChatSessions7d    int64 `json:"chat_sessions_7d"`
	ChatMessagesTotal int64 `json:"chat_messages_total"`
	HumanModeActive   int64 `json:"human_mode_active"`
	ContactsTotal     int64 `json:"contacts_total"`
	ContactsNew       int64 `json:"contacts_new"`
	KnowledgeCount    int64 `json:"knowledge_count"`
}
