package compatibility

// Result is the structured response from the compatibility check.
type Result struct {
	Compatible bool    `json:"compatible"`
	Issues     []Issue `json:"issues"`
	ItemCount  int     `json:"item_count"`
}

// Issue describes one compatibility problem found in the cart.
type Issue struct {
	// Type is "incompatible" (blocking) or "warning" (advisory).
	Type       string  `json:"type"`
	ProductIDs []int64 `json:"product_ids"`
	Message    string  `json:"message"`
}

// CartItem holds the product fields needed for compatibility analysis.
type CartItem struct {
	ProductID        int64
	Name             string
	ProductType      string // product_type column, e.g. "Выключатель", "Диммер"
	ConfiguratorType string // e.g. "switch", "dimmer", "frame", "socket"
	SeriesID         *int64
	SeriesName       string
	Quantity         int
}
