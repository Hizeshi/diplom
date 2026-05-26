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

// Result is returned after a completed import.
type Result struct {
	Total   int      `json:"total"`
	Upserted int     `json:"upserted"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
