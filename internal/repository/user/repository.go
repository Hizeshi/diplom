package userrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/compatibility"
	"iq-home/backend/internal/domain/user"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── Cart ────────────────────────────────────────────────────────────────────

func (r *Repository) GetCart(ctx context.Context, userID string) ([]user.CartItem, error) {
	const q = `
		SELECT ci.id, ci.product_id, p.name_raw, COALESCE(p.price, 0), ci.quantity,
		       COALESCE(pi.image_url, '')
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id AND p.deleted_at IS NULL
		LEFT JOIN LATERAL (
			SELECT image_url FROM product_images
			WHERE product_id = p.id ORDER BY display_order LIMIT 1
		) pi ON true
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC`

	return scanCartItems(r.db.Query(ctx, q, userID))
}

func (r *Repository) UpsertCartItem(ctx context.Context, userID string, productID int64, quantity int) error {
	const q = `
		INSERT INTO cart_items (user_id, product_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id) DO UPDATE
		SET quantity = EXCLUDED.quantity, updated_at = NOW()`

	_, err := r.db.Exec(ctx, q, userID, productID, quantity)
	if err != nil {
		return fmt.Errorf("userrepo: upsert cart: %w", err)
	}
	return nil
}

func (r *Repository) RemoveCartItem(ctx context.Context, userID string, productID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`,
		userID, productID)
	if err != nil {
		return fmt.Errorf("userrepo: remove cart item: %w", err)
	}
	return nil
}

// GetCartForCompatibility returns cart items enriched with product_type,
// configurator_type, series_id, and series name — fields needed for the
// AI compatibility check but not included in the regular cart response.
func (r *Repository) GetCartForCompatibility(ctx context.Context, userID string) ([]compatibility.CartItem, error) {
	const q = `
		SELECT ci.product_id,
		       p.name_raw,
		       COALESCE(p.product_type, ''),
		       COALESCE(p.configurator_type, ''),
		       p.series_id,
		       COALESCE(ps.name, ''),
		       ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id AND p.deleted_at IS NULL
		LEFT JOIN product_series ps ON ps.id = p.series_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get cart for compatibility: %w", err)
	}
	defer rows.Close()

	var items []compatibility.CartItem
	for rows.Next() {
		var it compatibility.CartItem
		if err := rows.Scan(
			&it.ProductID, &it.Name,
			&it.ProductType, &it.ConfiguratorType,
			&it.SeriesID, &it.SeriesName,
			&it.Quantity,
		); err != nil {
			return nil, fmt.Errorf("userrepo: scan cart compatibility: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ─── Favorites ───────────────────────────────────────────────────────────────

func (r *Repository) GetFavorites(ctx context.Context, userID string) ([]user.FavoriteItem, error) {
	const q = `
		SELECT f.product_id, p.name_raw, COALESCE(p.price, 0), COALESCE(pi.image_url, '')
		FROM favorites f
		JOIN products p ON p.id = f.product_id AND p.deleted_at IS NULL
		LEFT JOIN LATERAL (
			SELECT image_url FROM product_images
			WHERE product_id = p.id ORDER BY display_order LIMIT 1
		) pi ON true
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get favorites: %w", err)
	}
	defer rows.Close()

	var items []user.FavoriteItem
	for rows.Next() {
		var item user.FavoriteItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Price, &item.ImageURL); err != nil {
			return nil, fmt.Errorf("userrepo: scan favorite: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ToggleFavorite adds the product if not in favorites, removes it if it is.
func (r *Repository) ToggleFavorite(ctx context.Context, userID string, productID int64) error {
	// Atomic toggle: try to delete; if nothing was deleted — insert (only if product exists).
	const q = `
		WITH del AS (
			DELETE FROM favorites WHERE user_id = $1 AND product_id = $2 RETURNING product_id
		)
		INSERT INTO favorites (user_id, product_id)
		SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM del)
		  AND EXISTS (SELECT 1 FROM products WHERE id = $2 AND deleted_at IS NULL)`

	_, err := r.db.Exec(ctx, q, userID, productID)
	if err != nil {
		return fmt.Errorf("userrepo: toggle favorite: %w", err)
	}
	return nil
}

func (r *Repository) RemoveFavorite(ctx context.Context, userID string, productID int64) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM favorites WHERE user_id = $1 AND product_id = $2`,
		userID, productID)
	if err != nil {
		return fmt.Errorf("userrepo: remove favorite: %w", err)
	}
	return nil
}

// ─── History ─────────────────────────────────────────────────────────────────

func (r *Repository) GetHistory(ctx context.Context, userID string) ([]user.HistoryItem, error) {
	const q = `
		SELECT pv.product_id, p.name_raw, COALESCE(p.price, 0), COALESCE(pi.image_url, ''), pv.viewed_at
		FROM product_views pv
		JOIN products p ON p.id = pv.product_id AND p.deleted_at IS NULL
		LEFT JOIN LATERAL (
			SELECT image_url FROM product_images
			WHERE product_id = p.id ORDER BY display_order LIMIT 1
		) pi ON true
		WHERE pv.user_id = $1
		ORDER BY pv.viewed_at DESC
		LIMIT 20`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get history: %w", err)
	}
	defer rows.Close()

	var items []user.HistoryItem
	for rows.Next() {
		var item user.HistoryItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Price, &item.ImageURL, &item.ViewedAt); err != nil {
			return nil, fmt.Errorf("userrepo: scan history: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AddHistory(ctx context.Context, userID string, productID int64) error {
	// Use conditional insert to avoid FK violation for non-existent product_id.
	_, err := r.db.Exec(ctx,
		`INSERT INTO product_views (user_id, product_id)
		 SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM products WHERE id = $2 AND deleted_at IS NULL)`,
		userID, productID)
	if err != nil {
		return fmt.Errorf("userrepo: add history: %w", err)
	}

	// Keep only the 50 most recent views per user.
	_, err = r.db.Exec(ctx, `
		DELETE FROM product_views
		WHERE user_id = $1
		  AND id NOT IN (
			SELECT id FROM product_views
			WHERE user_id = $1
			ORDER BY viewed_at DESC
			LIMIT 50
		)`, userID)
	return err
}

func (r *Repository) GetRecommendations(ctx context.Context, userID string) ([]user.RecommendedItem, error) {
	const q = `
		WITH last_viewed AS (
			SELECT product_id FROM product_views
			WHERE user_id = $1
			ORDER BY viewed_at DESC
			LIMIT 1
		)
		SELECT p.id, p.name_raw, COALESCE(p.price, 0), COALESCE(pi.image_url, '')
		FROM product_vectors ref_vec
		JOIN product_vectors sim_vec
		  ON sim_vec.combined_embedding IS NOT NULL
		  AND sim_vec.product_id != ref_vec.product_id
		JOIN products p ON p.id = sim_vec.product_id AND p.deleted_at IS NULL AND p.is_active = true
		LEFT JOIN LATERAL (
			SELECT image_url FROM product_images
			WHERE product_id = p.id ORDER BY display_order LIMIT 1
		) pi ON true
		WHERE ref_vec.product_id = (SELECT product_id FROM last_viewed)
		ORDER BY sim_vec.combined_embedding <=> ref_vec.combined_embedding
		LIMIT 8`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get recommendations: %w", err)
	}
	defer rows.Close()

	var items []user.RecommendedItem
	for rows.Next() {
		var item user.RecommendedItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.ImageURL); err != nil {
			return nil, fmt.Errorf("userrepo: scan recommendation: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (r *Repository) GetOrders(ctx context.Context, userID string) ([]user.Order, error) {
	const q = `
		SELECT id, status, total_amount, payment_method, payment_status,
		       full_name, phone, address, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get orders: %w", err)
	}
	defer rows.Close()

	var orders []user.Order
	for rows.Next() {
		var o user.Order
		if err := rows.Scan(
			&o.ID, &o.Status, &o.TotalAmount, &o.PaymentMethod, &o.PaymentStatus,
			&o.FullName, &o.Phone, &o.Address, &o.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("userrepo: scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *Repository) GetOrderDetail(ctx context.Context, userID string, orderID int64) (*user.OrderDetail, error) {
	const q = `
		SELECT id, status, total_amount, payment_method, payment_status,
		       full_name, phone, address, created_at
		FROM orders
		WHERE id = $1 AND user_id = $2`

	var o user.OrderDetail
	err := r.db.QueryRow(ctx, q, orderID, userID).Scan(
		&o.ID, &o.Status, &o.TotalAmount, &o.PaymentMethod, &o.PaymentStatus,
		&o.FullName, &o.Phone, &o.Address, &o.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("userrepo: get order detail: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT product_id, product_name, price_at_purchase, quantity
		FROM order_items WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("userrepo: get order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item user.OrderItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.PriceAtPurchase, &item.Quantity); err != nil {
			return nil, fmt.Errorf("userrepo: scan order item: %w", err)
		}
		o.Items = append(o.Items, item)
	}
	return &o, rows.Err()
}

// Checkout creates an order from the user's cart inside a transaction.
func (r *Repository) Checkout(ctx context.Context, userID string, req user.CheckoutRequest) (int64, float64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("userrepo: checkout begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get cart items with current prices.
	rows, err := tx.Query(ctx, `
		SELECT ci.product_id, p.name_raw, p.price, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id AND p.deleted_at IS NULL
		WHERE ci.user_id = $1`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("userrepo: checkout get cart: %w", err)
	}

	type cartRow struct {
		productID int64
		name      string
		price     float64
		qty       int
	}
	var cartItems []cartRow
	var total float64

	for rows.Next() {
		var item cartRow
		if err := rows.Scan(&item.productID, &item.name, &item.price, &item.qty); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("userrepo: checkout scan cart: %w", err)
		}
		total += item.price * float64(item.qty)
		cartItems = append(cartItems, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	if len(cartItems) == 0 {
		return 0, 0, fmt.Errorf("cart is empty")
	}

	// 2. Create order.
	var orderID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, total_amount, payment_method, payment_status, full_name, phone, address)
		VALUES ($1, $2, $3, 'pending', $4, $5, $6)
		RETURNING id`,
		userID, total, req.PaymentMethod, req.Name, req.Phone, req.Address,
	).Scan(&orderID)
	if err != nil {
		return 0, 0, fmt.Errorf("userrepo: checkout create order: %w", err)
	}

	// 3. Insert order items.
	for _, item := range cartItems {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, price_at_purchase, quantity)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, item.productID, item.name, item.price, item.qty)
		if err != nil {
			return 0, 0, fmt.Errorf("userrepo: checkout insert item: %w", err)
		}
	}

	// 4. Clear cart.
	_, err = tx.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("userrepo: checkout clear cart: %w", err)
	}

	return orderID, total, tx.Commit(ctx)
}

// ─── Profile ─────────────────────────────────────────────────────────────────

func (r *Repository) GetProfile(ctx context.Context, userID string) (*user.Profile, error) {
	p := &user.Profile{ID: userID}
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(email, ''), COALESCE(full_name, ''), COALESCE(phone, ''),
		       COALESCE(avatar_url, ''), COALESCE(role, 'customer')
		FROM profiles WHERE id = $1`, userID).
		Scan(&p.Email, &p.FullName, &p.Phone, &p.AvatarURL, &p.Role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return p, nil // profile row not yet created — return partial with just ID
		}
		return nil, fmt.Errorf("userrepo: get profile: %w", err)
	}
	return p, nil
}

// ─── Session ─────────────────────────────────────────────────────────────────

func (r *Repository) GetSession(ctx context.Context, userID string) (*user.Session, error) {
	var s user.Session
	err := r.db.QueryRow(ctx, `
		SELECT session_id, is_human_mode
		FROM chat_sessions
		WHERE auth_user_id = $1
		ORDER BY updated_at DESC
		LIMIT 1`, userID).
		Scan(&s.SessionID, &s.IsHumanMode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("userrepo: get session: %w", err)
	}
	return &s, nil
}

// ─── Avatar ──────────────────────────────────────────────────────────────────

func (r *Repository) GetAvatarPath(ctx context.Context, userID string) (string, error) {
	var path string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(avatar_path, '') FROM profiles WHERE id = $1`, userID).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("userrepo: get avatar path: %w", err)
	}
	return path, nil
}

func (r *Repository) UpdateAvatar(ctx context.Context, userID, url, path string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE profiles SET avatar_url = $2, avatar_path = $3, updated_at = NOW() WHERE id = $1`,
		userID, url, path)
	if err != nil {
		return fmt.Errorf("userrepo: update avatar: %w", err)
	}
	return nil
}

func (r *Repository) ClearAvatar(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE profiles SET avatar_url = NULL, avatar_path = NULL, updated_at = NOW() WHERE id = $1`,
		userID)
	if err != nil {
		return fmt.Errorf("userrepo: clear avatar: %w", err)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanCartItems(rows pgx.Rows, err error) ([]user.CartItem, error) {
	if err != nil {
		return nil, fmt.Errorf("userrepo: get cart: %w", err)
	}
	defer rows.Close()

	var items []user.CartItem
	for rows.Next() {
		var item user.CartItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Name, &item.Price, &item.Quantity, &item.ImageURL); err != nil {
			return nil, fmt.Errorf("userrepo: scan cart item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
