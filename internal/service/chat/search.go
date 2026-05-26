package chatsvc

import (
	"context"
	"sync"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/domain/product"
)

// runSearch executes product search and knowledge search in parallel.
// If embedding is empty the vector-based searches are skipped gracefully.
func (s *Service) runSearch(
	ctx context.Context,
	query string,
	embedding []float32,
	matchCount int,
	topicFilter string,
) ([]chat.ProductMatch, []chat.KnowledgeMatch, error) {
	if len(embedding) == 0 {
		s.log.Warn("chat: skipping vector search — embedding is empty")
		return nil, nil, nil
	}
	var (
		products  []chat.ProductMatch
		knowledge []chat.KnowledgeMatch
		prodErr   error
		knowErr   error
		wg        sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		result, err := s.products.Search(ctx, product.SearchParams{
			Query:  query,
			Limit:  matchCount,
			Page:   1,
			SortBy: "relevance",
		}, embedding)
		if err != nil {
			prodErr = err
			return
		}
		products = toProductMatches(result)
	}()

	go func() {
		defer wg.Done()
		matches, err := s.repo.SearchKnowledge(ctx, embedding, 0.65, 5, topicFilter)
		if err != nil {
			knowErr = err
			return
		}
		knowledge = matches
	}()

	wg.Wait()

	// Product search error is non-fatal — return what we have.
	if prodErr != nil {
		s.log.Warn("chat: product search failed", "err", prodErr)
	}
	if knowErr != nil {
		s.log.Warn("chat: knowledge search failed", "err", knowErr)
	}

	return products, knowledge, nil
}

func toProductMatches(result *product.SearchResult) []chat.ProductMatch {
	if result == nil {
		return nil
	}
	matches := make([]chat.ProductMatch, 0, len(result.Items))
	for _, item := range result.Items {
		meta := map[string]any{
			"brand":  item.BrandName,
			"color":  item.ColorName,
			"series": item.SeriesName,
			"type":   item.Type,
		}
		if len(item.Images) > 0 {
			meta["image"] = item.Images[0].URL
		}
		matches = append(matches, chat.ProductMatch{
			ID:       item.ID,
			Name:     item.Name,
			Price:    item.Price,
			Score:    item.Score,
			Metadata: meta,
		})
	}
	return matches
}
