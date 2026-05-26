package paymentrepo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/payment"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UpdateOrderPaymentStatus applies a payment webhook result to an order.
// It is idempotent: if tx_id was already applied, returns ErrAlreadyHandled.
// Returns ErrOrderNotFound if no order with the given id exists.
func (r *Repository) UpdateOrderPaymentStatus(
	ctx context.Context,
	orderID int64,
	txID, paymentStatus, orderStatus string,
) error {
	// If tx_id is present, check whether this exact transaction was already processed.
	if txID != "" {
		var existing string
		err := r.db.QueryRow(ctx,
			`SELECT payment_tx_id FROM orders WHERE id = $1`,
			orderID,
		).Scan(&existing)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return payment.ErrOrderNotFound
		case err != nil:
			return err
		}
		if existing == txID {
			return payment.ErrAlreadyHandled
		}
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE orders
		    SET payment_status = $1,
		        status         = $2,
		        payment_tx_id  = $3,
		        updated_at     = NOW()
		  WHERE id = $4`,
		paymentStatus, orderStatus, txID, orderID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return payment.ErrOrderNotFound
	}
	return nil
}
