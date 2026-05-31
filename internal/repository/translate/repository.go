package translaterepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	translatesvc "iq-home/backend/internal/service/translate"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CountUntranslated(ctx context.Context, locale string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM products
		WHERE deleted_at IS NULL
		  AND (name_i18n->$1 IS NULL OR name_i18n->>$1 = '')`,
		locale,
	).Scan(&count)
	return count, err
}

func (r *Repository) FetchUntranslated(ctx context.Context, locale string, limit int) ([]translatesvc.ProductRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name_raw,
		       COALESCE(description, ''),
		       COALESCE(params_text, '')
		FROM products
		WHERE deleted_at IS NULL
		  AND (name_i18n->$1 IS NULL OR name_i18n->>$1 = '')
		ORDER BY id
		LIMIT $2`,
		locale, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("translaterepo: fetch: %w", err)
	}
	defer rows.Close()

	var result []translatesvc.ProductRow
	for rows.Next() {
		var p translatesvc.ProductRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Params); err != nil {
			return nil, fmt.Errorf("translaterepo: scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// SaveTranslations upserts translations for the given locale into JSONB columns.
func (r *Repository) SaveTranslations(ctx context.Context, locale string, items []translatesvc.TranslatedItem) error {
	for _, item := range items {
		_, err := r.db.Exec(ctx, `
			UPDATE products SET
				name_i18n        = name_i18n        || jsonb_build_object($2, $3::text),
				description_i18n = description_i18n || jsonb_build_object($2, $4::text),
				params_i18n      = params_i18n      || jsonb_build_object($2, $5::text),
				updated_at       = NOW()
			WHERE id = $1`,
			item.ID, locale, item.Name, item.Description, item.Params,
		)
		if err != nil {
			return fmt.Errorf("translaterepo: save id=%d: %w", item.ID, err)
		}
	}
	return nil
}
