package productimagerepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"iq-home/backend/internal/domain/productimage"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Insert saves a new product image record and returns its ID.
func (r *Repository) Insert(ctx context.Context, productID int, imageURL, bucket, objectPath string, displayOrder int) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`INSERT INTO product_images (product_id, image_url, bucket, object_path, display_order)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (product_id, bucket, object_path) DO UPDATE
		   SET image_url = EXCLUDED.image_url, display_order = EXCLUDED.display_order
		 RETURNING id`,
		productID, imageURL, bucket, objectPath, displayOrder,
	).Scan(&id)
	return id, err
}

// GetProductIDByArticle resolves a product ID by its article (case-insensitive).
// Returns pgx.ErrNoRows when no active product matches.
func (r *Repository) GetProductIDByArticle(ctx context.Context, article string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx,
		`SELECT id FROM products
		 WHERE UPPER(article) = UPPER($1) AND deleted_at IS NULL
		 LIMIT 1`,
		article,
	).Scan(&id)
	return id, err
}

// Get returns an image record by ID. Returns pgx.ErrNoRows when missing.
func (r *Repository) Get(ctx context.Context, id int) (*productimage.Image, error) {
	var img productimage.Image
	err := r.db.QueryRow(ctx,
		`SELECT id, product_id, image_url, bucket, object_path, display_order
		 FROM product_images WHERE id = $1`,
		id,
	).Scan(&img.ID, &img.ProductID, &img.ImageURL, &img.Bucket, &img.ObjectPath, &img.DisplayOrder)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// UpdateFile points an existing image record at a new storage object.
func (r *Repository) UpdateFile(ctx context.Context, id int, imageURL, bucket, objectPath string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE product_images SET image_url = $2, bucket = $3, object_path = $4 WHERE id = $1`,
		id, imageURL, bucket, objectPath,
	)
	return err
}

// UpdateOrder updates the display_order of an image by its ID.
func (r *Repository) UpdateOrder(ctx context.Context, id, displayOrder int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE product_images SET display_order = $1 WHERE id = $2`,
		displayOrder, id,
	)
	return err
}

// Delete removes an image record and returns its storage coordinates for cleanup.
func (r *Repository) Delete(ctx context.Context, id int) (*productimage.Image, error) {
	var img productimage.Image
	err := r.db.QueryRow(ctx,
		`DELETE FROM product_images WHERE id = $1 RETURNING id, product_id, image_url, bucket, object_path, display_order`,
		id,
	).Scan(&img.ID, &img.ProductID, &img.ImageURL, &img.Bucket, &img.ObjectPath, &img.DisplayOrder)
	if err != nil {
		return nil, err
	}
	return &img, nil
}
