package product

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
	ModelURL         string // URL to .glb 3D model for AR preview (empty if not available)
}

// Ref is a generic {id, name} reference used for brand/color/series.
type Ref struct {
	ID   int64
	Name string
}

// Image is a single product image.
type Image struct {
	ID   int64   `json:"id"`
	URL  string  `json:"url"`
	Path string  `json:"path"`
}

// SearchParams holds all filters and pagination for product search.
type SearchParams struct {
	Query    string
	Limit    int
	Page     int     // 1-based
	MinPrice *float64
	MaxPrice *float64
	BrandID  *int64
	ColorID  *int64
	Type     *string
	SeriesID *int64
	SortBy   string // "relevance" | "price_asc" | "price_desc"
}

// SearchResult is returned by the search endpoint.
type SearchResult struct {
	Items      []SearchItem `json:"items"`
	TotalCount int64        `json:"total"`
}

// SearchItem is one row from search_products().
type SearchItem struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Article          string  `json:"article"`
	Price            float64 `json:"price"`
	Score            float64 `json:"score"`
	Type             string  `json:"type"`
	ConfiguratorType string  `json:"configurator_type"`
	BrandName        string  `json:"brand"`
	ColorName        string  `json:"color"`
	SeriesName       string  `json:"series"`
	Images           []Image `json:"images"`
}

// Filters is returned by GET /api/filters.
type Filters struct {
	Types    []string  `json:"types"`
	Brands   []Option  `json:"brands"`
	Colors   []Option  `json:"colors"`
	Series   []Option  `json:"series"`
	MinPrice float64   `json:"min_price"`
	MaxPrice float64   `json:"max_price"`
}

// Option is a {id, name} pair used in filter lists.
type Option struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
