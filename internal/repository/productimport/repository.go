package productimportrepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/productimport"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a product by article.
// Brand, series, color are resolved by name inside the query.
func (r *Repository) Upsert(ctx context.Context, row productimport.Row) error {
	_, err := r.db.Exec(ctx, `
		WITH b AS (SELECT id FROM brands          WHERE name = $5 LIMIT 1),
		     s AS (SELECT id FROM product_series  WHERE name = $6 LIMIT 1),
		     c AS (SELECT id FROM colors          WHERE name = $7 LIMIT 1)
		INSERT INTO products (article, name_raw, price, product_type, brand_id, series_id, color_id, description)
		VALUES (
			$1,
			$2,
			$3::numeric,
			$4,
			(SELECT id FROM b),
			(SELECT id FROM s),
			(SELECT id FROM c),
			$8
		)
		ON CONFLICT (article) DO UPDATE SET
			name_raw     = EXCLUDED.name_raw,
			price        = EXCLUDED.price,
			product_type = EXCLUDED.product_type,
			description  = EXCLUDED.description,
			updated_at   = NOW()
	`,
		row.Article,
		row.Name,
		row.Price,
		row.ProductType,
		row.Brand,
		row.Series,
		row.Color,
		row.Description,
	)
	return err
}
