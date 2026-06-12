package productimage

import "errors"

// ErrNotFound is returned when the referenced image or product does not exist.
var ErrNotFound = errors.New("product image: not found")

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

// ReplaceRequest replaces the file of an existing image (admin "заменить фото").
type ReplaceRequest struct {
	ID          int
	Filename    string
	Data        []byte
	ContentType string
	// DisplayOrder is applied only when non-nil.
	DisplayOrder *int
}

// ─── Bulk upload report (contract with the admin panel) ─────────────────────

type BulkUploaded struct {
	Filename  string `json:"filename"`
	Article   string `json:"article"`
	ProductID int    `json:"product_id"`
}

type BulkSkipped struct {
	Filename string `json:"filename"`
	Article  string `json:"article"`
	Reason   string `json:"reason"`
}

type BulkError struct {
	Filename string `json:"filename"`
	Article  string `json:"article,omitempty"`
	Reason   string `json:"reason"`
}

// BulkReport is returned by the bulk upload endpoint. Slices are always
// non-nil so the admin panel can iterate without null checks.
type BulkReport struct {
	Uploaded []BulkUploaded `json:"uploaded"`
	Skipped  []BulkSkipped  `json:"skipped"`
	Errors   []BulkError    `json:"errors"`
}
