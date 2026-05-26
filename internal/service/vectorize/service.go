package vectorizesvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"iq-home/backend/internal/repository/vectorize"
)

type repo interface {
	ListProducts(ctx context.Context) ([]vectorizerepo.ProductRow, error)
	UpsertVector(ctx context.Context, productID int64, embedding []float32) error
}

type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Service struct {
	repo     repo
	embedder embedder
	log      *slog.Logger
}

func New(repo repo, embedder embedder, log *slog.Logger) *Service {
	return &Service{repo: repo, embedder: embedder, log: log}
}

type Result struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// VectorizeAll generates embeddings for all products and saves them to product_vectors.
func (s *Service) VectorizeAll(ctx context.Context) (*Result, error) {
	products, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("vectorize: list products: %w", err)
	}

	result := &Result{Total: len(products)}

	for _, p := range products {
		text := buildText(p)
		embedding, err := s.embedder.Embed(ctx, text)
		if err != nil {
			s.log.Warn("vectorize: embed failed", "product_id", p.ID, "err", err)
			result.Failed++
			continue
		}
		if err := s.repo.UpsertVector(ctx, p.ID, embedding); err != nil {
			s.log.Warn("vectorize: upsert failed", "product_id", p.ID, "err", err)
			result.Failed++
			continue
		}
		result.Success++
	}

	return result, nil
}

// buildText creates the combined text used for embedding a product.
func buildText(p vectorizerepo.ProductRow) string {
	parts := []string{p.NameRaw}
	if p.Article != "" {
		parts = append(parts, "Артикул: "+p.Article)
	}
	if p.Brand != "" {
		parts = append(parts, "Бренд: "+p.Brand)
	}
	if p.Series != "" {
		parts = append(parts, "Серия: "+p.Series)
	}
	if p.Color != "" {
		parts = append(parts, "Цвет: "+p.Color)
	}
	if p.ProductType != "" {
		parts = append(parts, "Тип: "+p.ProductType)
	}
	if p.Description != "" {
		parts = append(parts, p.Description)
	}
	return strings.Join(parts, ". ")
}
