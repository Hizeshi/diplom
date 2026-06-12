package productimport

// Row is one parsed row from the import file.
type Row struct {
	Article     string
	Name        string
	Price       float64
	ProductType string
	Brand       string
	Series      string
	Color       string
	Description string
}

// RowError describes a failed row in the import file.
type RowError struct {
	Row     int    `json:"row"`
	Article string `json:"article"`
	Error   string `json:"error"`
}

// Result is returned after a completed import.
// Upserted = Created + Updated (kept for backward compatibility).
type Result struct {
	Total    int        `json:"total"`
	Upserted int        `json:"upserted"`
	Created  int        `json:"created"`
	Updated  int        `json:"updated"`
	Failed   int        `json:"failed"`
	Errors   []RowError `json:"errors,omitempty"`
}
