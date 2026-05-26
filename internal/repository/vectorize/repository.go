package vectorizerepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRow is the minimal product data needed to build an embedding text.
type ProductRow struct {
	ID          int64
	NameRaw     string
	Article     string
	ProductType string
	Description string
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

// ListProducts returns all non-deleted products with their reference names.
func (r *Repository) ListProducts(ctx context.Context) ([]ProductRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			p.id,
			p.name_raw,
			COALESCE(p.article, ''),
			COALESCE(p.product_type, ''),
			COALESCE(p.description, ''),
			COALESCE(b.name, ''),
			COALESCE(s.name, ''),
			COALESCE(c.name, '')
		FROM products p
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN product_series s ON p.series_id = s.id
		LEFT JOIN colors c ON p.color_id = c.id
		WHERE p.deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProductRow
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(&p.ID, &p.NameRaw, &p.Article, &p.ProductType,
			&p.Description, &p.Brand, &p.Series, &p.Color); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// UpsertVector saves the combined embedding for a product.
func (r *Repository) UpsertVector(ctx context.Context, productID int64, embedding []float32) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO product_vectors (product_id, combined_embedding, updated_at)
		VALUES ($1, $2::vector, NOW())
		ON CONFLICT (product_id) DO UPDATE
		  SET combined_embedding = EXCLUDED.combined_embedding,
		      updated_at = NOW()
	`, productID, formatVector(embedding))
	return err
}

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
