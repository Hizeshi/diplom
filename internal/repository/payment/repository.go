package paymentrepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UpdateOrderPaymentStatus sets payment_status (and optionally order status) for an order.
func (r *Repository) UpdateOrderPaymentStatus(ctx context.Context, orderID int64, paymentStatus, orderStatus string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET payment_status = $1, status = $2, updated_at = NOW() WHERE id = $3`,
		paymentStatus, orderStatus, orderID,
	)
	return err
}
