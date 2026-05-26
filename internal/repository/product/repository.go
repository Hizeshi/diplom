package productrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/product"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func (r *Repository) GetByID(ctx context.Context, id int64) (*product.Product, error) {
	const q = `
		SELECT
			p.id, p.article, p.name_raw,
			COALESCE(p.product_type, ''),
			COALESCE(p.price, 0), COALESCE(p.description, ''),
			COALESCE(p.stock, 0), COALESCE(p.configurator_type, ''),
			b.id, b.name,
			c.id, c.name,
			s.id, s.name,
			COALESCE(p.model_url, '')
		FROM products p
		LEFT JOIN brands b         ON b.id = p.brand_id  AND b.deleted_at IS NULL
		LEFT JOIN colors c         ON c.id = p.color_id
		LEFT JOIN product_series s ON s.id = p.series_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	row := r.db.QueryRow(ctx, q, id)

	var (
		p                product.Product
		brandID, colorID, seriesID *int64
		brandName, colorName, seriesName *string
	)

	err := row.Scan(
		&p.ID, &p.Article, &p.Name, &p.Type,
		&p.Price, &p.Description, &p.Stock, &p.ConfiguratorType,
		&brandID, &brandName,
		&colorID, &colorName,
		&seriesID, &seriesName,
		&p.ModelURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("productrepo: get by id: %w", err)
	}

	if brandID != nil {
		p.Brand = &product.Ref{ID: *brandID, Name: strVal(brandName)}
	}
	if colorID != nil {
		p.Color = &product.Ref{ID: *colorID, Name: strVal(colorName)}
	}
	if seriesID != nil {
		p.Series = &product.Ref{ID: *seriesID, Name: strVal(seriesName)}
	}

	p.Images, err = r.imagesByProductID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) imagesByProductID(ctx context.Context, productID int64) ([]product.Image, error) {
	const q = `
		SELECT id, COALESCE(image_url, ''), COALESCE(object_path, '')
		FROM product_images
		WHERE product_id = $1
		ORDER BY display_order`

	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("productrepo: images: %w", err)
	}
	defer rows.Close()

	var images []product.Image
	for rows.Next() {
		var img product.Image
		if err := rows.Scan(&img.ID, &img.URL, &img.Path); err != nil {
			return nil, fmt.Errorf("productrepo: images scan: %w", err)
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// ─── Search ──────────────────────────────────────────────────────────────────

func (r *Repository) Search(ctx context.Context, params product.SearchParams, embedding []float32) (*product.SearchResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 12
	}
	offset := (max(params.Page, 1) - 1) * limit

	var vecStr *string
	if len(embedding) > 0 {
		s := formatVector(embedding)
		vecStr = &s
	}

	const q = `
		SELECT id, name_raw, COALESCE(price, 0), images, COALESCE(score, 0),
		       COALESCE(product_type, ''), COALESCE(configurator_type, ''),
		       COALESCE(brand_name, ''), COALESCE(color_name, ''), COALESCE(series_name, ''),
		       article, total_count
		FROM search_products(
			$1,              -- arg_query_text
			$2::vector,      -- arg_query_embedding
			0.15,            -- arg_match_threshold
			$3,              -- arg_page_limit
			$4,              -- arg_page_offset
			$5,              -- arg_filter_min_price
			$6,              -- arg_filter_max_price
			$7::integer,     -- arg_filter_brand_id
			$8::integer,     -- arg_filter_color_id
			$9,              -- arg_filter_product_type
			$10::integer,    -- arg_filter_series_id
			$11              -- arg_sort_by
		)`

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "relevance"
	}

	rows, err := r.db.Query(ctx, q,
		params.Query, vecStr,
		limit, offset,
		params.MinPrice, params.MaxPrice,
		params.BrandID, params.ColorID, params.Type, params.SeriesID,
		sortBy,
	)
	if err != nil {
		return nil, fmt.Errorf("productrepo: search: %w", err)
	}
	defer rows.Close()

	var (
		items      []product.SearchItem
		totalCount int64
	)

	for rows.Next() {
		var (
			item      product.SearchItem
			imagesRaw []byte
		)
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Price, &imagesRaw, &item.Score,
			&item.Type, &item.ConfiguratorType,
			&item.BrandName, &item.ColorName, &item.SeriesName, &item.Article,
			&totalCount,
		); err != nil {
			return nil, fmt.Errorf("productrepo: search scan: %w", err)
		}

		if len(imagesRaw) > 0 {
			_ = json.Unmarshal(imagesRaw, &item.Images)
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("productrepo: search rows: %w", err)
	}

	return &product.SearchResult{Items: items, TotalCount: totalCount}, nil
}

// ─── GetFilters ──────────────────────────────────────────────────────────────

func (r *Repository) GetFilters(ctx context.Context) (*product.Filters, error) {
	f := &product.Filters{}
	var err error

	// Types
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT product_type FROM products
		WHERE deleted_at IS NULL AND is_active = true AND product_type IS NOT NULL
		ORDER BY product_type`)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters types: %w", err)
	}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, err
		}
		f.Types = append(f.Types, t)
	}
	rows.Close()

	// Brands
	f.Brands, err = r.fetchOptions(ctx,
		`SELECT id, name FROM brands WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters brands: %w", err)
	}

	// Colors
	f.Colors, err = r.fetchOptions(ctx,
		`SELECT id, name FROM colors ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters colors: %w", err)
	}

	// Series
	f.Series, err = r.fetchOptions(ctx,
		`SELECT id, name FROM product_series ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters series: %w", err)
	}

	// Price range
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(MIN(price), 0), COALESCE(MAX(price), 0)
		FROM products WHERE deleted_at IS NULL AND is_active = true`).
		Scan(&f.MinPrice, &f.MaxPrice)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters price: %w", err)
	}

	return f, nil
}

func (r *Repository) fetchOptions(ctx context.Context, q string) ([]product.Option, error) {
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []product.Option
	for rows.Next() {
		var o product.Option
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func formatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
