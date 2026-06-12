package productimagesvc

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"iq-home/backend/internal/domain/productimage"
	"iq-home/backend/pkg/upload"
)

const bucket = "product-images"

type repo interface {
	Insert(ctx context.Context, productID int, imageURL, bucket, objectPath string, displayOrder int) (int, error)
	Get(ctx context.Context, id int) (*productimage.Image, error)
	GetProductIDByArticle(ctx context.Context, article string) (int, error)
	UpdateOrder(ctx context.Context, id, displayOrder int) error
	UpdateFile(ctx context.Context, id int, imageURL, bucket, objectPath string) error
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
	objectPath := fmt.Sprintf("%d/%s", req.ProductID, upload.UniqueFilename(req.Filename))

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

// BulkUpload matches each file to a product by the article encoded in its
// filename ("АРТИКУЛ.jpg" or "АРТИКУЛ-1.jpg"; a purely numeric name is treated
// as a product ID for backward compatibility) and returns a per-file report.
func (s *Service) BulkUpload(ctx context.Context, items []productimage.BulkUploadItem) (*productimage.BulkReport, error) {
	report := &productimage.BulkReport{
		Uploaded: []productimage.BulkUploaded{},
		Skipped:  []productimage.BulkSkipped{},
		Errors:   []productimage.BulkError{},
	}

	for _, item := range items {
		article, productID, err := s.resolveProduct(ctx, item.Filename)
		if err != nil {
			report.Skipped = append(report.Skipped, productimage.BulkSkipped{
				Filename: item.Filename,
				Article:  article,
				Reason:   "товар с таким артикулом не найден",
			})
			continue
		}

		objectPath := fmt.Sprintf("%d/%s", productID, upload.UniqueFilename(item.Filename))
		url, err := s.storage.Upload(ctx, bucket, objectPath, item.Data, item.ContentType)
		if err != nil {
			report.Errors = append(report.Errors, productimage.BulkError{
				Filename: item.Filename,
				Article:  article,
				Reason:   fmt.Sprintf("ошибка загрузки в хранилище: %v", err),
			})
			continue
		}

		if _, err := s.repo.Insert(ctx, productID, url, bucket, objectPath, item.DisplayOrder); err != nil {
			report.Errors = append(report.Errors, productimage.BulkError{
				Filename: item.Filename,
				Article:  article,
				Reason:   fmt.Sprintf("ошибка записи в БД: %v", err),
			})
			continue
		}

		report.Uploaded = append(report.Uploaded, productimage.BulkUploaded{
			Filename:  item.Filename,
			Article:   article,
			ProductID: productID,
		})
	}
	return report, nil
}

// resolveProduct maps a filename to (article, productID).
// Tries the full base name as an article, then the base without a trailing
// "-N"/"_N" counter suffix; a purely numeric base is used as a product ID.
func (s *Service) resolveProduct(ctx context.Context, filename string) (string, int, error) {
	base := strings.TrimSuffix(filename, path.Ext(filename))

	candidates := []string{base}
	if i := strings.LastIndexAny(base, "-_"); i > 0 {
		if _, err := strconv.Atoi(base[i+1:]); err == nil {
			candidates = append(candidates, base[:i])
		}
	}

	for _, article := range candidates {
		if id, err := s.repo.GetProductIDByArticle(ctx, article); err == nil {
			return article, id, nil
		}
	}

	// Legacy: filenames like "1234.jpg" or "1234_front.jpg" carry the product ID.
	idPart := base
	if i := strings.IndexByte(base, '_'); i >= 0 {
		idPart = base[:i]
	}
	if id, err := strconv.Atoi(idPart); err == nil && id > 0 {
		return base, id, nil
	}

	return base, 0, productimage.ErrNotFound
}

// UpdateOrder changes the display order of an existing image.
func (s *Service) UpdateOrder(ctx context.Context, id, displayOrder int) error {
	return s.repo.UpdateOrder(ctx, id, displayOrder)
}

// Replace uploads a new file for an existing image record, updates the record
// and removes the old object from storage (best-effort).
func (s *Service) Replace(ctx context.Context, req productimage.ReplaceRequest) (*productimage.Image, error) {
	img, err := s.repo.Get(ctx, req.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, productimage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("productimage: replace get: %w", err)
	}

	objectPath := fmt.Sprintf("%d/%s", img.ProductID, upload.UniqueFilename(req.Filename))
	url, err := s.storage.Upload(ctx, bucket, objectPath, req.Data, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("productimage: replace upload: %w", err)
	}

	if err := s.repo.UpdateFile(ctx, req.ID, url, bucket, objectPath); err != nil {
		return nil, fmt.Errorf("productimage: replace update: %w", err)
	}
	if req.DisplayOrder != nil {
		if err := s.repo.UpdateOrder(ctx, req.ID, *req.DisplayOrder); err != nil {
			return nil, fmt.Errorf("productimage: replace order: %w", err)
		}
		img.DisplayOrder = *req.DisplayOrder
	}

	// Best-effort cleanup of the previous object.
	if img.ObjectPath != "" && img.ObjectPath != objectPath {
		_ = s.storage.Delete(ctx, img.Bucket, []string{img.ObjectPath})
	}

	img.ImageURL = url
	img.Bucket = bucket
	img.ObjectPath = objectPath
	return img, nil
}

// Delete removes the image from storage and the DB.
func (s *Service) Delete(ctx context.Context, id int) error {
	img, err := s.repo.Delete(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return productimage.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("productimage: delete db: %w", err)
	}
	// Best-effort storage cleanup
	_ = s.storage.Delete(ctx, img.Bucket, []string{img.ObjectPath})
	return nil
}
