package quote

// Request is the payload for POST /v1/quotes.
type Request struct {
	CompanyName string `json:"company_name"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Items       []Item `json:"items"`
	Note        string `json:"note"`
}

// Item is a single line in the quote.
type Item struct {
	Name     string  `json:"name"`
	Article  string  `json:"article"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"` // unit price
}

// Response is returned after successful PDF generation.
type Response struct {
	URL string `json:"url"` // public Supabase Storage URL
}
