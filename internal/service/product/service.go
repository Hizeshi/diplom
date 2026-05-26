package productsvc

import (
	"context"
	"fmt"

	"iq-home/backend/internal/domain/product"
)

// repo is the subset of productrepo.Repository the service needs.
type repo interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
	Search(ctx context.Context, params product.SearchParams, embedding []float32) (*product.SearchResult, error)
	GetFilters(ctx context.Context) (*product.Filters, error)
}

// embedder generates vector embeddings (satisfied by client/openai.EmbedAdapter).
type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Service struct {
	repo    repo
	embedder embedder
}

func New(repo repo, embedder embedder) *Service {
	return &Service{repo: repo, embedder: embedder}
}

// GetByID returns a single product with all its relations and images.
// Returns nil, nil when the product does not exist.
func (s *Service) GetByID(ctx context.Context, id int64) (*product.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("product service: get by id: %w", err)
	}
	return p, nil
}

// Search runs a semantic + full-text hybrid search.
// When params.Query is empty it falls back to a filter-only query (no embedding).
func (s *Service) Search(ctx context.Context, params product.SearchParams) (*product.SearchResult, error) {
	var embedding []float32

	if params.Query != "" {
		var err error
		embedding, err = s.embedder.Embed(ctx, params.Query)
		if err != nil {
			// Non-fatal: fall back to text-only search
			embedding = nil
		}
	}

	result, err := s.repo.Search(ctx, params, embedding)
	if err != nil {
		return nil, fmt.Errorf("product service: search: %w", err)
	}
	return result, nil
}

// GetFilters returns all available filter options and price range.
func (s *Service) GetFilters(ctx context.Context) (*product.Filters, error) {
	f, err := s.repo.GetFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("product service: get filters: %w", err)
	}
	return f, nil
}
