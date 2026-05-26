package productimagesvc

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"iq-home/backend/internal/domain/productimage"
)

const bucket = "product-images"

type repo interface {
	Insert(ctx context.Context, productID int, imageURL, bucket, objectPath string, displayOrder int) (int, error)
	UpdateOrder(ctx context.Context, id, displayOrder int) error
	Delete(ctx context.Context, id int) (*productimage.Image, error)
}

type storage interface {
	Upload(ctx context.Context, bucket, objectPath string, data []byte, contentType string) (string, error)
	Delete(ctx context.Context, bucket string, paths []string) error
	PublicURL(bucket, objectPath string) string
}

type Service struct {
	repo    repo
	storage storage
}

func New(repo repo, storage storage) *Service {
	return &Service{repo: repo, storage: storage}
}

// Add uploads one image to Supabase Storage and inserts the DB record.
func (s *Service) Add(ctx context.Context, req productimage.AddRequest) (*productimage.Image, error) {
	objectPath := fmt.Sprintf("%d/%s", req.ProductID, req.Filename)

	url, err := s.storage.Upload(ctx, bucket, objectPath, req.Data, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("productimage: upload: %w", err)
	}

	id, err := s.repo.Insert(ctx, req.ProductID, url, bucket, objectPath, req.DisplayOrder)
	if err != nil {
		return nil, fmt.Errorf("productimage: insert: %w", err)
	}

	return &productimage.Image{
		ID:           id,
		ProductID:    req.ProductID,
		ImageURL:     url,
		Bucket:       bucket,
		ObjectPath:   objectPath,
		DisplayOrder: req.DisplayOrder,
	}, nil
}

// BulkUpload parses product ID from each filename ("1234_name.jpg" → productID=1234).
func (s *Service) BulkUpload(ctx context.Context, items []productimage.BulkUploadItem) ([]*productimage.Image, error) {
	var results []*productimage.Image
	for _, item := range items {
		productID, err := parseProductIDFromFilename(item.Filename)
		if err != nil {
			continue // skip files with unrecognised names
		}

		objectPath := fmt.Sprintf("%d/%s", productID, item.Filename)
		url, err := s.storage.Upload(ctx, bucket, objectPath, item.Data, item.ContentType)
		if err != nil {
			continue // non-fatal per item
		}

		id, err := s.repo.Insert(ctx, productID, url, bucket, objectPath, item.DisplayOrder)
		if err != nil {
			continue
		}
		results = append(results, &productimage.Image{
			ID:           id,
			ProductID:    productID,
			ImageURL:     url,
			Bucket:       bucket,
			ObjectPath:   objectPath,
			DisplayOrder: item.DisplayOrder,
		})
	}
	return results, nil
}

// UpdateOrder changes the display order of an existing image.
func (s *Service) UpdateOrder(ctx context.Context, id, displayOrder int) error {
	return s.repo.UpdateOrder(ctx, id, displayOrder)
}

// Delete removes the image from storage and the DB.
func (s *Service) Delete(ctx context.Context, id int) error {
	img, err := s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("productimage: delete db: %w", err)
	}
	// Best-effort storage cleanup
	_ = s.storage.Delete(ctx, img.Bucket, []string{img.ObjectPath})
	return nil
}

// parseProductIDFromFilename extracts the leading numeric segment before "_" or ".".
// E.g. "1234_front.jpg" → 1234, "1234.jpg" → 1234.
func parseProductIDFromFilename(filename string) (int, error) {
	base := strings.TrimSuffix(filename, path.Ext(filename))
	part := base
	if idx := strings.IndexByte(base, '_'); idx >= 0 {
		part = base[:idx]
	}
	return strconv.Atoi(part)
}
