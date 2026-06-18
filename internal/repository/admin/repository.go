package adminrepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/admin"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── Chats ───────────────────────────────────────────────────────────────────

func (r *Repository) ListChats(ctx context.Context) ([]admin.ChatSession, error) {
	// Uses the admin_chat_list view defined in schema.
	rows, err := r.db.Query(ctx, `
		SELECT session_id, is_human_mode,
		       TO_CHAR(last_active AT TIME ZONE 'UTC+5', 'DD.MM HH24:MI'),
		       COALESCE(last_message, '')
		FROM admin_chat_list
		LIMIT 50`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list chats: %w", err)
	}
	defer rows.Close()

	var sessions []admin.ChatSession
	for rows.Next() {
		var s admin.ChatSession
		if err := rows.Scan(&s.SessionID, &s.IsHumanMode, &s.LastActive, &s.LastMessage); err != nil {
			return nil, fmt.Errorf("adminrepo: scan chat session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) GetChatHistory(ctx context.Context, sessionID string) ([]admin.ChatMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT role, content, COALESCE(sender_type, ''), created_at
		FROM chat_messages
		WHERE session_id = $1
		ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: get chat history: %w", err)
	}
	defer rows.Close()

	var msgs []admin.ChatMessage
	for rows.Next() {
		var m admin.ChatMessage
		if err := rows.Scan(&m.Role, &m.Content, &m.SenderType, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("adminrepo: scan chat message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *Repository) ToggleHumanMode(ctx context.Context, sessionID string, enabled bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE chat_sessions SET is_human_mode = $2, updated_at = NOW() WHERE session_id = $1`,
		sessionID, enabled)
	if err != nil {
		return fmt.Errorf("adminrepo: toggle human mode: %w", err)
	}
	return nil
}

func (r *Repository) SendManagerMessage(ctx context.Context, sessionID, text string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chat_messages (session_id, role, content, sender_type, message_type)
		VALUES ($1, 'assistant', $2, 'manager', 'text')`,
		sessionID, text)
	if err != nil {
		return fmt.Errorf("adminrepo: send manager message: %w", err)
	}
	_, _ = r.db.Exec(ctx, `UPDATE chat_sessions SET updated_at = NOW() WHERE session_id = $1`, sessionID)
	return nil
}

// ─── Products ────────────────────────────────────────────────────────────────

func (r *Repository) ListProducts(ctx context.Context, search string, limit, page int) (*admin.ProductList, error) {
	if limit <= 0 {
		limit = 20
	}
	offset := (max(page, 1) - 1) * limit

	args := []any{limit, offset}
	where := "p.deleted_at IS NULL"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (p.name_raw ILIKE $%d OR p.article ILIKE $%d)", len(args), len(args))
	}

	q := fmt.Sprintf(`
		SELECT p.id, p.article, p.name_raw, COALESCE(p.product_type,''), COALESCE(p.price,0), COALESCE(p.stock,0),
		       COALESCE(p.description,''), p.brand_id, COALESCE(b.name,''),
		       p.series_id, COALESCE(s.name,''), p.color_id, COALESCE(c.name,''),
		       COALESCE(p.configurator_type,''), COALESCE(p.is_active,true), p.created_at,
		       COUNT(*) OVER() AS total,
		       pi.id, pi.image_url, pi.object_path
		FROM products p
		LEFT JOIN brands b ON b.id = p.brand_id AND b.deleted_at IS NULL
		LEFT JOIN product_series s ON s.id = p.series_id
		LEFT JOIN colors c ON c.id = p.color_id
		LEFT JOIN LATERAL (
			SELECT id, COALESCE(image_url,''), COALESCE(object_path,'')
			FROM product_images WHERE product_id = p.id
			ORDER BY display_order LIMIT 1
		) pi(id, image_url, object_path) ON true
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list products: %w", err)
	}
	defer rows.Close()

	var (
		items []admin.Product
		total int64
	)
	for rows.Next() {
		var p admin.Product
		var imgID *int64
		var imgURL, imgPath *string
		if err := rows.Scan(
			&p.ID, &p.Article, &p.Name, &p.Type, &p.Price, &p.Stock,
			&p.Description, &p.BrandID, &p.BrandName,
			&p.SeriesID, &p.SeriesName, &p.ColorID, &p.ColorName,
			&p.ConfiguratorType, &p.IsActive, &p.CreatedAt, &total,
			&imgID, &imgURL, &imgPath,
		); err != nil {
			return nil, fmt.Errorf("adminrepo: scan product: %w", err)
		}
		if imgID != nil {
			p.Images = []admin.ProductImage{{ID: *imgID, URL: *imgURL, Path: *imgPath}}
		} else {
			p.Images = []admin.ProductImage{}
		}
		items = append(items, p)
	}
	return &admin.ProductList{Items: items, Total: total}, rows.Err()
}

func (r *Repository) CreateProduct(ctx context.Context, data admin.ProductCreate) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO products (name_raw, article, price, stock, product_type,
		                      description, brand_id, series_id, color_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true)
		RETURNING id`,
		data.Name, data.Article, data.Price, data.Stock, data.Type,
		data.Description, data.BrandID, data.SeriesID, data.ColorID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("adminrepo: create product: %w", err)
	}
	return id, nil
}

func (r *Repository) UpdateProduct(ctx context.Context, id int64, data admin.ProductUpdate) error {
	_, err := r.db.Exec(ctx, `
		UPDATE products SET
			name_raw = $2, article = $3, price = $4, stock = $5,
			description = $6, product_type = $7,
			brand_id = $8, series_id = $9, color_id = $10,
			configurator_type = $11, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`,
		id, data.Name, data.Article, data.Price, data.Stock,
		data.Description, data.Type,
		data.BrandID, data.SeriesID, data.ColorID,
		data.ConfiguratorType)
	if err != nil {
		return fmt.Errorf("adminrepo: update product: %w", err)
	}
	return nil
}

func (r *Repository) DeleteProduct(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE products SET deleted_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("adminrepo: delete product: %w", err)
	}
	return nil
}

func (r *Repository) ScanDuplicates(ctx context.Context) ([]admin.DuplicatePair, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p1.id, p1.name_raw, p2.id, p2.name_raw,
		       1 - (pv1.combined_embedding <=> pv2.combined_embedding) AS similarity
		FROM product_vectors pv1
		JOIN product_vectors pv2
		  ON pv2.product_id > pv1.product_id
		  AND (1 - (pv1.combined_embedding <=> pv2.combined_embedding)) > 0.99
		JOIN products p1 ON p1.id = pv1.product_id AND p1.deleted_at IS NULL
		JOIN products p2 ON p2.id = pv2.product_id AND p2.deleted_at IS NULL
		ORDER BY similarity DESC
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: scan duplicates: %w", err)
	}
	defer rows.Close()

	var pairs []admin.DuplicatePair
	for rows.Next() {
		var p admin.DuplicatePair
		if err := rows.Scan(&p.ID, &p.Name, &p.DuplicateID, &p.DuplicateName, &p.Similarity); err != nil {
			return nil, fmt.Errorf("adminrepo: scan duplicate: %w", err)
		}
		pairs = append(pairs, p)
	}
	return pairs, rows.Err()
}

// ─── Users ───────────────────────────────────────────────────────────────────

func (r *Repository) ListUsers(ctx context.Context, search string, limit, page int) (*admin.UserList, error) {
	if limit <= 0 {
		limit = 20
	}
	offset := (max(page, 1) - 1) * limit

	args := []any{limit, offset}
	where := "1=1"
	if search != "" {
		args = append(args, "%"+search+"%")
		idx := fmt.Sprintf("$%d", len(args))
		where = fmt.Sprintf("(p.full_name ILIKE %s OR p.email ILIKE %s OR p.phone ILIKE %s)", idx, idx, idx)
	}

	q := fmt.Sprintf(`
		SELECT p.id::text, COALESCE(p.email,''), COALESCE(p.full_name,''),
		       COALESCE(p.phone,''), COALESCE(p.role,'customer'),
		       COALESCE(p.avatar_url,''), p.created_at,
		       COUNT(o.id) AS orders_count,
		       COALESCE(SUM(o.total_amount), 0) AS total_spent,
		       COUNT(*) OVER() AS total
		FROM profiles p
		LEFT JOIN orders o ON o.user_id::uuid = p.id
		WHERE %s
		GROUP BY p.id, p.email, p.full_name, p.phone, p.role, p.avatar_url, p.created_at
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`, where)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list users: %w", err)
	}
	defer rows.Close()

	var (
		items []admin.User
		total int64
	)
	for rows.Next() {
		var u admin.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.AvatarURL, &u.CreatedAt,
			&u.OrdersCount, &u.TotalSpent, &total,
		); err != nil {
			return nil, fmt.Errorf("adminrepo: scan user: %w", err)
		}
		u.RegisteredAt = u.CreatedAt
		items = append(items, u)
	}
	return &admin.UserList{Items: items, Total: total}, rows.Err()
}

func (r *Repository) GetUserDetail(ctx context.Context, id string) (*admin.UserDetail, error) {
	var u admin.UserDetail
	err := r.db.QueryRow(ctx, `
		SELECT id::text, COALESCE(email,''), COALESCE(full_name,''),
		       COALESCE(phone,''), COALESCE(role,'customer'),
		       COALESCE(avatar_url,''), created_at
		FROM profiles WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.AvatarURL, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("adminrepo: get user: %w", err)
	}

	// Orders
	rows, err := r.db.Query(ctx,
		`SELECT id, status, total_amount, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: user orders: %w", err)
	}
	for rows.Next() {
		var o admin.UserOrder
		if err := rows.Scan(&o.ID, &o.Status, &o.TotalAmount, &o.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		u.Orders = append(u.Orders, o)
	}
	rows.Close()

	// Cart
	rows, err = r.db.Query(ctx, `
		SELECT ci.product_id, p.name_raw, COALESCE(p.price,0), ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: user cart: %w", err)
	}
	for rows.Next() {
		var c admin.UserCartItem
		if err := rows.Scan(&c.ProductID, &c.Name, &c.Price, &c.Quantity); err != nil {
			rows.Close()
			return nil, err
		}
		u.Cart = append(u.Cart, c)
	}
	rows.Close()

	// History
	rows, err = r.db.Query(ctx, `
		SELECT pv.product_id, p.name_raw, pv.viewed_at
		FROM product_views pv
		JOIN products p ON p.id = pv.product_id
		WHERE pv.user_id = $1
		ORDER BY pv.viewed_at DESC LIMIT 20`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: user history: %w", err)
	}
	for rows.Next() {
		var h admin.UserHistoryItem
		if err := rows.Scan(&h.ProductID, &h.Name, &h.ViewedAt); err != nil {
			rows.Close()
			return nil, err
		}
		u.History = append(u.History, h)
	}
	rows.Close()

	// Favorites
	rows, err = r.db.Query(ctx, `
		SELECT f.product_id, p.name_raw, COALESCE(p.price,0)
		FROM favorites f
		JOIN products p ON p.id = f.product_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: user favorites: %w", err)
	}
	for rows.Next() {
		var f admin.UserFavoriteItem
		if err := rows.Scan(&f.ProductID, &f.Name, &f.Price); err != nil {
			rows.Close()
			return nil, err
		}
		u.Favorites = append(u.Favorites, f)
	}
	rows.Close()

	return &u, nil
}

func (r *Repository) UpdateUser(ctx context.Context, id string, data admin.UserUpdate) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET full_name = $2, phone = $3, avatar_url = $4, updated_at = NOW()
		WHERE id = $1`, id, data.FullName, data.Phone, data.AvatarURL)
	if err != nil {
		return fmt.Errorf("adminrepo: update user: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUserData(ctx context.Context, id string) error {
	// Cascades handle related records; profiles row references auth.users.
	_, err := r.db.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("adminrepo: delete user data: %w", err)
	}
	return nil
}

func (r *Repository) UpdateUserRole(ctx context.Context, id, role string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE profiles SET role = $2, updated_at = NOW() WHERE id = $1`, id, role)
	if err != nil {
		return fmt.Errorf("adminrepo: update user role: %w", err)
	}
	return nil
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (r *Repository) ListOrders(ctx context.Context) ([]admin.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, status, total_amount, payment_method, payment_status,
		       full_name, phone, address, created_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list orders: %w", err)
	}
	defer rows.Close()

	var orders []admin.Order
	for rows.Next() {
		var o admin.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.PaymentMethod, &o.PaymentStatus,
			&o.FullName, &o.Phone, &o.Address, &o.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("adminrepo: scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *Repository) GetOrderDetail(ctx context.Context, id int64) (*admin.OrderDetail, error) {
	var o admin.OrderDetail
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount, payment_method, payment_status,
		       full_name, phone, address, created_at
		FROM orders WHERE id = $1`, id).
		Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.PaymentMethod, &o.PaymentStatus,
			&o.FullName, &o.Phone, &o.Address, &o.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("adminrepo: get order: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT product_id, product_name, price_at_purchase, quantity FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: get order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item admin.OrderItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.PriceAtPurchase, &item.Quantity); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, item)
	}
	return &o, rows.Err()
}

// ─── Knowledge ───────────────────────────────────────────────────────────────

func (r *Repository) ListKnowledge(ctx context.Context) ([]admin.KnowledgeEntry, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, topic, content, created_at FROM sales_knowledge ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list knowledge: %w", err)
	}
	defer rows.Close()

	var entries []admin.KnowledgeEntry
	for rows.Next() {
		var e admin.KnowledgeEntry
		if err := rows.Scan(&e.ID, &e.Topic, &e.Content, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("adminrepo: scan knowledge: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *Repository) UpsertKnowledge(ctx context.Context, id *int64, topic, content string, embedding []float32) error {
	vecStr := formatVector(embedding)

	if id == nil {
		_, err := r.db.Exec(ctx,
			`INSERT INTO sales_knowledge (topic, content, embedding) VALUES ($1, $2, $3::vector)`,
			topic, content, vecStr)
		if err != nil {
			return fmt.Errorf("adminrepo: insert knowledge: %w", err)
		}
	} else {
		_, err := r.db.Exec(ctx,
			`UPDATE sales_knowledge SET topic = $2, content = $3, embedding = $4::vector WHERE id = $1`,
			*id, topic, content, vecStr)
		if err != nil {
			return fmt.Errorf("adminrepo: update knowledge: %w", err)
		}
	}
	return nil
}

func (r *Repository) DeleteKnowledge(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sales_knowledge WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("adminrepo: delete knowledge: %w", err)
	}
	return nil
}

// ─── Contacts ────────────────────────────────────────────────────────────────

func (r *Repository) ListContacts(ctx context.Context) ([]admin.Contact, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, email, message, status, created_at FROM contact_requests ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []admin.Contact
	for rows.Next() {
		var c admin.Contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Message, &c.Status, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("adminrepo: scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// ─── Metadata ────────────────────────────────────────────────────────────────

func (r *Repository) GetMetadata(ctx context.Context) (*admin.Metadata, error) {
	m := &admin.Metadata{}

	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM products      WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM profiles),
			(SELECT COUNT(*) FROM orders),
			(SELECT COALESCE(SUM(total_amount), 0) FROM orders),
			(SELECT COUNT(*) FROM sales_knowledge)
	`).Scan(&m.ProductsTotal, &m.UsersTotal, &m.OrdersTotal, &m.OrdersRevenue, &m.KnowledgeCount)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: metadata: %w", err)
	}

	return m, nil
}

// GetStats собирает детальную статистику для дашборда админки.
func (r *Repository) GetStats(ctx context.Context) (*admin.Stats, error) {
	s := &admin.Stats{}

	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM products WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM products WHERE deleted_at IS NULL AND is_active = true),
			(SELECT COUNT(*) FROM products p WHERE p.deleted_at IS NULL
				AND NOT EXISTS (SELECT 1 FROM product_images pi WHERE pi.product_id = p.id)),
			(SELECT COUNT(*) FROM products WHERE deleted_at IS NULL AND is_active = true AND COALESCE(stock, 0) <= 5),
			(SELECT COUNT(*) FROM profiles),
			(SELECT COUNT(*) FROM profiles WHERE created_at >= now() - INTERVAL '30 days'),
			(SELECT COUNT(*) FROM orders),
			(SELECT COUNT(*) FROM orders WHERE created_at >= now() - INTERVAL '30 days'),
			(SELECT COALESCE(SUM(total_amount), 0) FROM orders),
			(SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE payment_status = 'success'),
			(SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE created_at >= now() - INTERVAL '30 days'),
			(SELECT COALESCE(AVG(total_amount), 0) FROM orders),
			(SELECT COUNT(*) FROM chat_sessions),
			(SELECT COUNT(*) FROM chat_sessions WHERE updated_at >= now() - INTERVAL '7 days'),
			(SELECT COUNT(*) FROM chat_messages),
			(SELECT COUNT(*) FROM chat_sessions WHERE is_human_mode = true),
			(SELECT COUNT(*) FROM contact_requests),
			(SELECT COUNT(*) FROM contact_requests WHERE status = 'new'),
			(SELECT COUNT(*) FROM sales_knowledge)
	`).Scan(
		&s.ProductsTotal, &s.ProductsActive, &s.ProductsNoImages, &s.ProductsLowStock,
		&s.UsersTotal, &s.UsersNew30d,
		&s.OrdersTotal, &s.Orders30d,
		&s.RevenueTotal, &s.RevenuePaid, &s.Revenue30d, &s.AvgOrderValue,
		&s.ChatSessionsTotal, &s.ChatSessions7d, &s.ChatMessagesTotal, &s.HumanModeActive,
		&s.ContactsTotal, &s.ContactsNew, &s.KnowledgeCount,
	)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: stats totals: %w", err)
	}

	s.OrdersByStatus, err = r.fetchStatusCounts(ctx,
		`SELECT status, COUNT(*) FROM orders GROUP BY status ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: stats by status: %w", err)
	}

	s.PaymentMethods, err = r.fetchStatusCounts(ctx,
		`SELECT payment_method, COUNT(*) FROM orders GROUP BY payment_method ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: stats payment methods: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT d::date::text,
		       COUNT(o.id),
		       COALESCE(SUM(o.total_amount), 0)
		FROM generate_series(CURRENT_DATE - INTERVAL '13 days', CURRENT_DATE, '1 day') d
		LEFT JOIN orders o ON o.created_at::date = d::date
		GROUP BY d
		ORDER BY d`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: stats daily: %w", err)
	}
	defer rows.Close()
	s.OrdersDaily = []admin.DailyOrders{}
	for rows.Next() {
		var d admin.DailyOrders
		if err := rows.Scan(&d.Date, &d.Count, &d.Revenue); err != nil {
			return nil, fmt.Errorf("adminrepo: stats daily scan: %w", err)
		}
		s.OrdersDaily = append(s.OrdersDaily, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("adminrepo: stats daily rows: %w", err)
	}

	topRows, err := r.db.Query(ctx, `
		SELECT COALESCE(oi.product_id, 0),
		       COALESCE(oi.product_name, '—'),
		       COALESCE(SUM(oi.quantity), 0),
		       COALESCE(SUM(oi.price_at_purchase * oi.quantity), 0)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id AND o.status <> 'cancelled'
		GROUP BY oi.product_id, oi.product_name
		ORDER BY SUM(oi.price_at_purchase * oi.quantity) DESC NULLS LAST
		LIMIT 5`)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: stats top products: %w", err)
	}
	defer topRows.Close()
	s.TopProducts = []admin.TopProduct{}
	for topRows.Next() {
		var t admin.TopProduct
		if err := topRows.Scan(&t.ProductID, &t.Name, &t.Quantity, &t.Revenue); err != nil {
			return nil, fmt.Errorf("adminrepo: stats top scan: %w", err)
		}
		s.TopProducts = append(s.TopProducts, t)
	}
	if err := topRows.Err(); err != nil {
		return nil, fmt.Errorf("adminrepo: stats top rows: %w", err)
	}

	return s, nil
}

func (r *Repository) fetchStatusCounts(ctx context.Context, q string) ([]admin.StatusCount, error) {
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []admin.StatusCount{}
	for rows.Next() {
		var sc admin.StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (r *Repository) fetchOptions(ctx context.Context, q string) ([]admin.Option, error) {
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []admin.Option
	for rows.Next() {
		var o admin.Option
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

// ─── User Data Management ─────────────────────────────────────────────────────

func (r *Repository) ClearUserCart(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) ClearUserHistory(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_views WHERE user_id = $1`, userID)
	return err
}

// ─── Products Extra ───────────────────────────────────────────────────────────

func (r *Repository) UpdateConfiguratorType(ctx context.Context, productID int64, configuratorType string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE products SET configurator_type = $1, updated_at = NOW() WHERE id = $2`,
		configuratorType, productID,
	)
	return err
}

// ─── Products single ─────────────────────────────────────────────────────────

func (r *Repository) GetProduct(ctx context.Context, id int64) (*admin.Product, error) {
	var p admin.Product
	err := r.db.QueryRow(ctx, `
		SELECT p.id, p.article, p.name_raw, COALESCE(p.product_type,''),
		       COALESCE(p.price,0), COALESCE(p.stock,0), COALESCE(p.description,''),
		       p.brand_id, COALESCE(b.name,''),
		       p.series_id, COALESCE(s.name,''),
		       p.color_id, COALESCE(c.name,''),
		       COALESCE(p.configurator_type,''), COALESCE(p.is_active,true), p.created_at
		FROM products p
		LEFT JOIN brands b ON b.id = p.brand_id
		LEFT JOIN product_series s ON s.id = p.series_id
		LEFT JOIN colors c ON c.id = p.color_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`, id).
		Scan(&p.ID, &p.Article, &p.Name, &p.Type,
			&p.Price, &p.Stock, &p.Description,
			&p.BrandID, &p.BrandName,
			&p.SeriesID, &p.SeriesName,
			&p.ColorID, &p.ColorName,
			&p.ConfiguratorType, &p.IsActive, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("adminrepo: get product: %w", err)
	}

	imgRows, err := r.db.Query(ctx,
		`SELECT id, COALESCE(image_url,''), COALESCE(object_path,'')
		 FROM product_images WHERE product_id = $1 ORDER BY display_order`, id)
	if err != nil {
		return nil, fmt.Errorf("adminrepo: get product images: %w", err)
	}
	defer imgRows.Close()
	for imgRows.Next() {
		var img admin.ProductImage
		if err := imgRows.Scan(&img.ID, &img.URL, &img.Path); err != nil {
			return nil, fmt.Errorf("adminrepo: scan product image: %w", err)
		}
		p.Images = append(p.Images, img)
	}
	if p.Images == nil {
		p.Images = []admin.ProductImage{}
	}

	return &p, nil
}

// ─── Orders management ───────────────────────────────────────────────────────

func (r *Repository) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("adminrepo: update order status: %w", err)
	}
	return nil
}

func (r *Repository) DeleteOrder(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM orders WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("adminrepo: delete order: %w", err)
	}
	return nil
}

// ─── User item management ────────────────────────────────────────────────────

func (r *Repository) DeleteCartItem(ctx context.Context, userID string, productID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *Repository) DeleteHistoryItem(ctx context.Context, userID string, productID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM product_views WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *Repository) DeleteFavoriteItem(ctx context.Context, userID string, productID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM favorites WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *Repository) ClearUserFavorites(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1`, userID)
	return err
}

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
