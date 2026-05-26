package productsvc_test

import (
	"context"
	"errors"
	"testing"

	"iq-home/backend/internal/domain/product"
	productsvc "iq-home/backend/internal/service/product"
)

// ─── Mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	product *product.Product
	result  *product.SearchResult
	filters *product.Filters
	err     error
	// capture args
	lastSearchEmbedding []float32
}

func (m *mockRepo) GetByID(_ context.Context, _ int64) (*product.Product, error) {
	return m.product, m.err
}

func (m *mockRepo) Search(_ context.Context, _ product.SearchParams, emb []float32) (*product.SearchResult, error) {
	m.lastSearchEmbedding = emb
	return m.result, m.err
}

func (m *mockRepo) GetFilters(_ context.Context) (*product.Filters, error) {
	return m.filters, m.err
}

type mockEmbedder struct {
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return m.vec, m.err
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestGetByID_ReturnsProduct(t *testing.T) {
	want := &product.Product{ID: 42, Name: "Умная лампа"}
	svc := productsvc.New(&mockRepo{product: want}, &mockEmbedder{})

	got, err := svc.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != want.ID {
		t.Errorf("expected product id=42, got %v", got)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := productsvc.New(&mockRepo{product: nil}, &mockEmbedder{})

	got, err := svc.GetByID(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil product, got %v", got)
	}
}

func TestGetByID_RepoError(t *testing.T) {
	svc := productsvc.New(&mockRepo{err: errors.New("db down")}, &mockEmbedder{})

	_, err := svc.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Search ───────────────────────────────────────────────────────────────────

func TestSearch_WithQuery_EmbeddingPassed(t *testing.T) {
	vec := []float32{0.1, 0.2, 0.3}
	repo := &mockRepo{result: &product.SearchResult{}}
	svc := productsvc.New(repo, &mockEmbedder{vec: vec})

	_, err := svc.Search(context.Background(), product.SearchParams{Query: "умная лампа"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastSearchEmbedding) == 0 {
		t.Error("expected embedding to be passed to repo, got empty")
	}
}

func TestSearch_EmptyQuery_NoEmbedding(t *testing.T) {
	repo := &mockRepo{result: &product.SearchResult{}}
	svc := productsvc.New(repo, &mockEmbedder{vec: []float32{1, 2, 3}})

	_, err := svc.Search(context.Background(), product.SearchParams{Query: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastSearchEmbedding) != 0 {
		t.Error("expected no embedding for empty query")
	}
}

func TestSearch_EmbedFails_FallsBackToTextSearch(t *testing.T) {
	// Embedder fails — search must still proceed without embedding
	repo := &mockRepo{result: &product.SearchResult{}}
	svc := productsvc.New(repo, &mockEmbedder{err: errors.New("openai down")})

	_, err := svc.Search(context.Background(), product.SearchParams{Query: "лампа"})
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if len(repo.lastSearchEmbedding) != 0 {
		t.Error("expected nil embedding after embedder failure")
	}
}

func TestSearch_RepoError(t *testing.T) {
	svc := productsvc.New(
		&mockRepo{err: errors.New("db error")},
		&mockEmbedder{},
	)
	_, err := svc.Search(context.Background(), product.SearchParams{})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// ─── GetFilters ───────────────────────────────────────────────────────────────

func TestGetFilters_ReturnsFilters(t *testing.T) {
	want := &product.Filters{MinPrice: 1000, MaxPrice: 99000}
	svc := productsvc.New(&mockRepo{filters: want}, &mockEmbedder{})

	got, err := svc.GetFilters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MinPrice != want.MinPrice || got.MaxPrice != want.MaxPrice {
		t.Errorf("expected filters %+v, got %+v", want, got)
	}
}
