package describerepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRow contains the data needed to generate a description.
type ProductRow struct {
	ID          int64
	NameRaw     string
	Article     string
	ProductType string
	Brand       string
	Series      string
	Color       string
}

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ListProducts returns all non-deleted products (without their current description).
func (r *Repository) ListProducts(ctx context.Context) ([]ProductRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			p.id,
			p.name_raw,
			COALESCE(p.article, ''),
			COALESCE(p.product_type, ''),
			COALESCE(b.name, ''),
			COALESCE(s.name, ''),
			COALESCE(c.name, '')
		FROM products p
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN product_series s ON p.series_id = s.id
		LEFT JOIN colors c ON p.color_id = c.id
		WHERE p.deleted_at IS NULL
		ORDER BY p.id
	`)
	if err != nil {
		return nil, fmt.Errorf("describerepo: list: %w", err)
	}
	defer rows.Close()

	var result []ProductRow
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(&p.ID, &p.NameRaw, &p.Article, &p.ProductType,
			&p.Brand, &p.Series, &p.Color); err != nil {
			return nil, fmt.Errorf("describerepo: scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// UpdateDescription saves a generated description for a product.
func (r *Repository) UpdateDescription(ctx context.Context, productID int64, description string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE products SET description = $2, updated_at = NOW() WHERE id = $1`,
		productID, description)
	if err != nil {
		return fmt.Errorf("describerepo: update: %w", err)
	}
	return nil
}
