package productimage

// Image represents a product image record.
type Image struct {
	ID           int    `json:"id"`
	ProductID    int    `json:"product_id"`
	ImageURL     string `json:"image_url"`
	DisplayOrder int    `json:"display_order"`
	Bucket       string `json:"bucket"`
	ObjectPath   string `json:"object_path"`
}

// BulkUploadItem is a single file in a bulk upload request.
type BulkUploadItem struct {
	ProductID    int    // parsed from filename
	Filename     string // original filename
	Data         []byte
	ContentType  string
	DisplayOrder int
}

// AddRequest is used for adding a single image via API.
type AddRequest struct {
	ProductID    int
	Filename     string
	Data         []byte
	ContentType  string
	DisplayOrder int
}

// UpdateRequest updates display order of an existing image.
type UpdateRequest struct {
	ID           int `json:"id"`
	DisplayOrder int `json:"display_order"`
}
