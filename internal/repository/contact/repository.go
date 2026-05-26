package contactrepo

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

func (r *Repository) Create(ctx context.Context, name, email, message string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO contact_requests (name, email, message) VALUES ($1, $2, $3)`,
		name, email, message,
	)
	return err
}
