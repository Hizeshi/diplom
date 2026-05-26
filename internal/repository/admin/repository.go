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
		SELECT p.id, p.article, p.name_raw, COALESCE(p.product_type,''), p.price, p.stock,
		       COALESCE(p.description,''), p.brand_id, COALESCE(b.name,''),
		       p.series_id, COALESCE(s.name,''), p.color_id, COALESCE(c.name,''),
		       COALESCE(p.configurator_type,''), p.is_active, p.created_at,
		       COUNT(*) OVER() AS total
		FROM products p
		LEFT JOIN brands b ON b.id = p.brand_id AND b.deleted_at IS NULL
		LEFT JOIN product_series s ON s.id = p.series_id
		LEFT JOIN colors c ON c.id = p.color_id
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
		if err := rows.Scan(
			&p.ID, &p.Article, &p.Name, &p.Type, &p.Price, &p.Stock,
			&p.Description, &p.BrandID, &p.BrandName,
			&p.SeriesID, &p.SeriesName, &p.ColorID, &p.ColorName,
			&p.ConfiguratorType, &p.IsActive, &p.CreatedAt, &total,
		); err != nil {
			return nil, fmt.Errorf("adminrepo: scan product: %w", err)
		}
		items = append(items, p)
	}
	return &admin.ProductList{Items: items, Total: total}, rows.Err()
}

func (r *Repository) CreateProduct(ctx context.Context, data admin.ProductCreate) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO products (name_raw, article, price, stock, product_type, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id`,
		data.Name, data.Article, data.Price, data.Stock, data.Type,
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
		       COUNT(*) OVER() AS total
		FROM profiles p
		WHERE %s
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
			&u.ID, &u.Email, &u.FullName, &u.Phone, &u.Role, &u.AvatarURL, &u.CreatedAt, &total,
		); err != nil {
			return nil, fmt.Errorf("adminrepo: scan user: %w", err)
		}
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
		SELECT ci.product_id, p.name_raw, p.price, ci.quantity
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
