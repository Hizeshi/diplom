package productsvc

import (
	"context"
	"fmt"

	"iq-home/backend/internal/domain/product"
)

type repo interface {
	GetByID(ctx context.Context, id int64, locale string) (*product.Product, error)
	Search(ctx context.Context, params product.SearchParams, embedding []float32) (*product.SearchResult, error)
	GetFilters(ctx context.Context, locale string) (*product.Filters, error)
	UpdateProductI18n(ctx context.Context, id int64, locale, name, description, params string) error
	UpdateSeriesI18n(ctx context.Context, id int64, locale, name string) error
	UpdateBrandI18n(ctx context.Context, id int64, locale, name string) error
	UpdateColorI18n(ctx context.Context, id int64, locale, name string) error
}

type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type Service struct {
	repo     repo
	embedder embedder
}

func New(repo repo, embedder embedder) *Service {
	return &Service{repo: repo, embedder: embedder}
}

func (s *Service) GetByID(ctx context.Context, id int64, locale string) (*product.Product, error) {
	p, err := s.repo.GetByID(ctx, id, locale)
	if err != nil {
		return nil, fmt.Errorf("product service: get by id: %w", err)
	}
	return p, nil
}

func (s *Service) Search(ctx context.Context, params product.SearchParams) (*product.SearchResult, error) {
	var embedding []float32
	if params.Query != "" {
		var err error
		embedding, err = s.embedder.Embed(ctx, params.Query)
		if err != nil {
			embedding = nil
		}
	}
	result, err := s.repo.Search(ctx, params, embedding)
	if err != nil {
		return nil, fmt.Errorf("product service: search: %w", err)
	}
	return result, nil
}

func (s *Service) GetFilters(ctx context.Context, locale string) (*product.Filters, error) {
	f, err := s.repo.GetFilters(ctx, locale)
	if err != nil {
		return nil, fmt.Errorf("product service: get filters: %w", err)
	}
	return f, nil
}

func (s *Service) UpdateProductI18n(ctx context.Context, id int64, upd product.I18nUpdate) error {
	return s.repo.UpdateProductI18n(ctx, id, upd.Locale, upd.Name, upd.Description, upd.Params)
}

func (s *Service) UpdateSeriesI18n(ctx context.Context, id int64, upd product.I18nUpdate) error {
	return s.repo.UpdateSeriesI18n(ctx, id, upd.Locale, upd.Name)
}

func (s *Service) UpdateBrandI18n(ctx context.Context, id int64, upd product.I18nUpdate) error {
	return s.repo.UpdateBrandI18n(ctx, id, upd.Locale, upd.Name)
}

func (s *Service) UpdateColorI18n(ctx context.Context, id int64, upd product.I18nUpdate) error {
	return s.repo.UpdateColorI18n(ctx, id, upd.Locale, upd.Name)
}
