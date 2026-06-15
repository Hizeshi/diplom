package productrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/domain/product"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func (r *Repository) GetByID(ctx context.Context, id int64, locale string) (*product.Product, error) {
	const q = `
		SELECT
			p.id, p.article,
			p.name_raw,        p.name_i18n,
			COALESCE(p.product_type, ''),
			COALESCE(p.price, 0),
			COALESCE(p.description, ''), p.description_i18n,
			COALESCE(p.stock, 0),
			COALESCE(p.configurator_type, ''),
			b.id, b.name, b.name_i18n,
			c.id, c.name, c.name_i18n,
			s.id, s.name, s.name_i18n,
			COALESCE(p.model_url, '')
		FROM products p
		LEFT JOIN brands b         ON b.id = p.brand_id  AND b.deleted_at IS NULL
		LEFT JOIN colors c         ON c.id = p.color_id
		LEFT JOIN product_series s ON s.id = p.series_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	row := r.db.QueryRow(ctx, q, id)

	var (
		p                           product.Product
		nameI18n, descI18n          product.I18nMap
		brandID, colorID, seriesID  *int64
		brandName, colorName, seriesName *string
		brandI18n, colorI18n, seriesI18n product.I18nMap
	)

	err := row.Scan(
		&p.ID, &p.Article,
		&p.Name, &nameI18n,
		&p.Type,
		&p.Price,
		&p.Description, &descI18n,
		&p.Stock, &p.ConfiguratorType,
		&brandID, &brandName, &brandI18n,
		&colorID, &colorName, &colorI18n,
		&seriesID, &seriesName, &seriesI18n,
		&p.ModelURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("productrepo: get by id: %w", err)
	}

	p.Name = product.Localize(p.Name, nameI18n, locale)
	p.Description = product.Localize(p.Description, descI18n, locale)

	if brandID != nil {
		p.Brand = &product.Ref{ID: *brandID, Name: product.Localize(strVal(brandName), brandI18n, locale)}
	}
	if colorID != nil {
		p.Color = &product.Ref{ID: *colorID, Name: product.Localize(strVal(colorName), colorI18n, locale)}
	}
	if seriesID != nil {
		p.Series = &product.Ref{ID: *seriesID, Name: product.Localize(strVal(seriesName), seriesI18n, locale)}
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

// ─── GetByIDs ────────────────────────────────────────────────────────────────

// GetByIDs fetches a lightweight representation of products by their IDs,
// suitable for injecting into the chat prompt context.
func (r *Repository) GetByIDs(ctx context.Context, ids []int64) ([]chat.ProductMatch, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name_raw, COALESCE(p.article, ''), COALESCE(p.price, 0),
		       COALESCE(b.name, ''), COALESCE(c.name, ''), COALESCE(s.name, '')
		FROM products p
		LEFT JOIN brands b         ON b.id = p.brand_id
		LEFT JOIN colors c         ON c.id = p.color_id
		LEFT JOIN product_series s ON s.id = p.series_id
		WHERE p.id = ANY($1) AND p.deleted_at IS NULL`, ids)
	if err != nil {
		return nil, fmt.Errorf("productrepo: get by ids: %w", err)
	}
	defer rows.Close()

	var matches []chat.ProductMatch
	for rows.Next() {
		var (
			m                    chat.ProductMatch
			article              string
			brand, color, series string
		)
		if err := rows.Scan(&m.ID, &m.Name, &article, &m.Price, &brand, &color, &series); err != nil {
			return nil, fmt.Errorf("productrepo: get by ids scan: %w", err)
		}
		m.Metadata = map[string]any{"brand": brand, "color": color, "series": series, "article": article}
		matches = append(matches, m)
	}
	return matches, rows.Err()
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
			$1,          -- arg_query_text
			$2::vector,  -- arg_query_embedding
			0.15,        -- arg_match_threshold
			$3,          -- arg_page_limit
			$4,          -- arg_page_offset
			$5,          -- arg_filter_min_price
			$6,          -- arg_filter_max_price
			$7::integer, -- arg_filter_brand_id
			$8::integer, -- arg_filter_color_id
			$9,          -- arg_filter_product_type
			$10::integer,-- arg_filter_series_id
			$11          -- arg_sort_by
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
			item                        product.SearchItem
			imagesRaw                   []byte
			brandName, colorName, seriesName string
		)
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Price, &imagesRaw, &item.Score,
			&item.Type, &item.ConfiguratorType,
			&brandName, &colorName, &seriesName, &item.Article,
			&totalCount,
		); err != nil {
			return nil, fmt.Errorf("productrepo: search scan: %w", err)
		}
		if len(imagesRaw) > 0 {
			_ = json.Unmarshal(imagesRaw, &item.Images)
		}
		if brandName != "" {
			item.Brand = &product.NameRef{Name: brandName}
		}
		if colorName != "" {
			item.Color = &product.NameRef{Name: colorName}
		}
		if seriesName != "" {
			item.Series = &product.NameRef{Name: seriesName}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("productrepo: search rows: %w", err)
	}

	// Apply localization via a single batch query (avoids N+1).
	if len(items) > 0 && params.Locale != "" && params.Locale != product.LocaleRU {
		if err := r.localizeSearchItems(ctx, items, params.Locale); err != nil {
			// Non-fatal: log and return unlocalized results.
			_ = err
		}
	}

	return &product.SearchResult{Items: items, TotalCount: totalCount}, nil
}

// localizeSearchItems fetches i18n data for a batch of search results and applies translations.
func (r *Repository) localizeSearchItems(ctx context.Context, items []product.SearchItem, locale string) error {
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name_i18n,
		       COALESCE(b.name_i18n, '{}'), COALESCE(c.name_i18n, '{}'), COALESCE(s.name_i18n, '{}')
		FROM products p
		LEFT JOIN brands b         ON b.id = p.brand_id
		LEFT JOIN colors c         ON c.id = p.color_id
		LEFT JOIN product_series s ON s.id = p.series_id
		WHERE p.id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		name, brand, color, series product.I18nMap
	}
	i18nByID := make(map[int64]row, len(ids))
	for rows.Next() {
		var id int64
		var r row
		if err := rows.Scan(&id, &r.name, &r.brand, &r.color, &r.series); err != nil {
			return err
		}
		i18nByID[id] = r
	}

	for i := range items {
		if d, ok := i18nByID[items[i].ID]; ok {
			items[i].Name = product.Localize(items[i].Name, d.name, locale)
			if items[i].Brand != nil {
				items[i].Brand.Name = product.Localize(items[i].Brand.Name, d.brand, locale)
			}
			if items[i].Color != nil {
				items[i].Color.Name = product.Localize(items[i].Color.Name, d.color, locale)
			}
			if items[i].Series != nil {
				items[i].Series.Name = product.Localize(items[i].Series.Name, d.series, locale)
			}
		}
	}
	return nil
}

// ─── GetFilters ──────────────────────────────────────────────────────────────

func (r *Repository) GetFilters(ctx context.Context, locale string) (*product.Filters, error) {
	f := &product.Filters{}
	var err error

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
		f.Types = append(f.Types, localizeType(t, locale))
	}
	rows.Close()

	f.Brands, err = r.fetchOptionsI18n(ctx,
		`SELECT id, name, name_i18n FROM brands WHERE deleted_at IS NULL ORDER BY name`, locale)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters brands: %w", err)
	}

	f.Colors, err = r.fetchOptionsI18n(ctx,
		`SELECT id, name, name_i18n FROM colors ORDER BY name`, locale)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters colors: %w", err)
	}

	f.Series, err = r.fetchSeriesOptions(ctx, locale)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters series: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(MIN(price), 0), COALESCE(MAX(price), 0)
		FROM products WHERE deleted_at IS NULL AND is_active = true`).
		Scan(&f.MinPrice, &f.MaxPrice)
	if err != nil {
		return nil, fmt.Errorf("productrepo: filters price: %w", err)
	}

	return f, nil
}

func (r *Repository) fetchOptionsI18n(ctx context.Context, q, locale string) ([]product.Option, error) {
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []product.Option
	for rows.Next() {
		var (
			o    product.Option
			i18n product.I18nMap
		)
		if err := rows.Scan(&o.ID, &o.Name, &i18n); err != nil {
			return nil, err
		}
		o.Name = product.Localize(o.Name, i18n, locale)
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

func (r *Repository) fetchSeriesOptions(ctx context.Context, locale string) ([]product.SeriesOption, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, name_i18n, brand_id FROM product_series ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opts []product.SeriesOption
	for rows.Next() {
		var (
			o    product.SeriesOption
			i18n product.I18nMap
		)
		if err := rows.Scan(&o.ID, &o.Name, &i18n, &o.BrandID); err != nil {
			return nil, err
		}
		o.Name = product.Localize(o.Name, i18n, locale)
		opts = append(opts, o)
	}
	return opts, rows.Err()
}

// ─── UpdateI18n ───────────────────────────────────────────────────────────────

func (r *Repository) UpdateProductI18n(ctx context.Context, id int64, locale, name, description, params string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE products SET
			name_i18n        = jsonb_set(COALESCE(name_i18n,'{}'),        $2, $3::jsonb),
			description_i18n = jsonb_set(COALESCE(description_i18n,'{}'), $2, $4::jsonb),
			params_i18n      = jsonb_set(COALESCE(params_i18n,'{}'),      $2, $5::jsonb),
			updated_at       = NOW()
		WHERE id = $1 AND deleted_at IS NULL`,
		id, []string{locale},
		jsonStr(name), jsonStr(description), jsonStr(params),
	)
	return err
}

func (r *Repository) UpdateSeriesI18n(ctx context.Context, id int64, locale, name string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE product_series SET name_i18n = jsonb_set(COALESCE(name_i18n,'{}'), $2, $3::jsonb)
		WHERE id = $1`,
		id, []string{locale}, jsonStr(name),
	)
	return err
}

func (r *Repository) UpdateBrandI18n(ctx context.Context, id int64, locale, name string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name_i18n = jsonb_set(COALESCE(name_i18n,'{}'), $2, $3::jsonb)
		WHERE id = $1 AND deleted_at IS NULL`,
		id, []string{locale}, jsonStr(name),
	)
	return err
}

func (r *Repository) UpdateColorI18n(ctx context.Context, id int64, locale, name string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE colors SET name_i18n = jsonb_set(COALESCE(name_i18n,'{}'), $2, $3::jsonb)
		WHERE id = $1`,
		id, []string{locale}, jsonStr(name),
	)
	return err
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

var typeTranslations = map[string]map[string]string{
	"Выключатели":             {"kk": "Ажыратқыштар",          "en": "Switches"},
	"Диммеры, Светорегулятор": {"kk": "Диммерлер, жарық реттегіш", "en": "Dimmers"},
	"Заглушки / Вывод кабеля": {"kk": "Тығындар / Кабель шығысы",  "en": "Blanks / Cable outlet"},
	"Рамки":                   {"kk": "Жақтаулар",             "en": "Frames"},
	"Розетка":                 {"kk": "Розетка",               "en": "Socket"},
	"Розетки компьютерные":    {"kk": "Компьютерлік розеткалар", "en": "Computer sockets"},
	"Розетки телевизионные":   {"kk": "Теледидар розеткалары", "en": "TV sockets"},
	"Розетки телефонные":      {"kk": "Телефон розеткалары",   "en": "Phone sockets"},
	"Розетки электрические":   {"kk": "Электр розеткалары",    "en": "Electrical sockets"},
}

func localizeType(t, locale string) string {
	if locale == "ru" || locale == "" {
		return t
	}
	if m, ok := typeTranslations[t]; ok {
		if v, ok := m[locale]; ok && v != "" {
			return v
		}
	}
	return t
}

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
