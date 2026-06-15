package product

import "encoding/json"

// I18nMap holds translations keyed by locale ("kk", "en", "ru").
// Example: {"kk": "Ажыратқыш", "en": "Switch"}
type I18nMap map[string]string

// Scan implements pgx Scanner for JSONB columns.
func (m *I18nMap) Scan(src any) error {
	if src == nil {
		*m = I18nMap{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	}
	if len(b) == 0 || string(b) == "{}" {
		*m = I18nMap{}
		return nil
	}
	return json.Unmarshal(b, m)
}

// Localize returns the best translation for the given locale.
// Priority: requested locale → "ru" entry in i18n → raw fallback.
func Localize(raw string, i18n I18nMap, locale string) string {
	if i18n == nil {
		return raw
	}
	if v, ok := i18n[locale]; ok && v != "" {
		return v
	}
	if locale != LocaleRU {
		if v, ok := i18n[LocaleRU]; ok && v != "" {
			return v
		}
	}
	return raw
}

const (
	LocaleRU = "ru"
	LocaleKK = "kk"
	LocaleEN = "en"
)

// ─── Product ──────────────────────────────────────────────────────────────────

// Product is the full product detail (used for GET /api/products/:id).
type Product struct {
	ID               int64
	Article          string
	Name             string
	Type             string
	Price            float64
	Description      string
	Stock            int
	ConfiguratorType string
	Brand            *Ref
	Color            *Ref
	Series           *Ref
	Images           []Image
}

// Ref is a generic {id, name} reference used for brand/color/series.
type Ref struct {
	ID   int64
	Name string
}

// NameRef is a {name} reference used for brand/color/series in search results
// (search_products() returns names only, not IDs).
type NameRef struct {
	Name string `json:"name"`
}

// Image is a single product image.
type Image struct {
	ID   int64  `json:"id"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

// ─── Search ───────────────────────────────────────────────────────────────────

// SearchParams holds all filters and pagination for product search.
type SearchParams struct {
	Query    string
	Limit    int
	Page     int
	MinPrice *float64
	MaxPrice *float64
	BrandID  *int64
	ColorID  *int64
	Type     *string
	SeriesID *int64
	SortBy   string
	Locale   string // "ru" | "kk" | "en"
}

// SearchResult is returned by the search endpoint.
type SearchResult struct {
	Items      []SearchItem `json:"items"`
	TotalCount int64        `json:"total"`
}

// SearchItem is one row from search_products().
type SearchItem struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Article          string   `json:"article"`
	Price            float64  `json:"price"`
	Score            float64  `json:"score"`
	Type             string   `json:"type"`
	ConfiguratorType string   `json:"configurator_type"`
	Brand            *NameRef `json:"brand"`
	Color            *NameRef `json:"color"`
	Series           *NameRef `json:"series"`
	Images           []Image  `json:"images"`
}

// ─── Filters ──────────────────────────────────────────────────────────────────

// Filters is returned by GET /api/filters.
type Filters struct {
	Types    []string       `json:"types"`
	Brands   []Option       `json:"brands"`
	Colors   []Option       `json:"colors"`
	Series   []SeriesOption `json:"series"`
	MinPrice float64        `json:"min_price"`
	MaxPrice float64        `json:"max_price"`
}

// Option is a {id, name} pair used in filter lists.
type Option struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// SeriesOption extends Option with brand_id for the configurator.
type SeriesOption struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	BrandID *int64 `json:"brand_id"`
}

// ─── I18n admin ───────────────────────────────────────────────────────────────

// I18nUpdate is the body for PUT /api/admin/{resource}/{id}/i18n
type I18nUpdate struct {
	Locale      string `json:"locale"      validate:"required,oneof=ru kk en"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      string `json:"params"`
}
