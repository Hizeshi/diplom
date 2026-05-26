package describesvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	describerepo "iq-home/backend/internal/repository/describe"
	"iq-home/backend/internal/client/openai"
)

type repo interface {
	ListProducts(ctx context.Context) ([]describerepo.ProductRow, error)
	UpdateDescription(ctx context.Context, productID int64, description string) error
}

type llm interface {
	Complete(ctx context.Context, opts openai.CompleteOptions) (string, error)
}

type Service struct {
	repo  repo
	llm   llm
	model string
	log   *slog.Logger
}

func New(repo repo, llm llm, model string, log *slog.Logger) *Service {
	return &Service{repo: repo, llm: llm, model: model, log: log}
}

type Result struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// DescribeAll generates descriptions for all products using OpenAI.
func (s *Service) DescribeAll(ctx context.Context) (*Result, error) {
	products, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("describe: list products: %w", err)
	}

	result := &Result{Total: len(products)}

	for _, p := range products {
		desc, err := s.generateDescription(ctx, p)
		if err != nil {
			s.log.Warn("describe: generate failed", "product_id", p.ID, "err", err)
			result.Failed++
			continue
		}
		if err := s.repo.UpdateDescription(ctx, p.ID, desc); err != nil {
			s.log.Warn("describe: update failed", "product_id", p.ID, "err", err)
			result.Failed++
			continue
		}
		result.Success++
	}

	return result, nil
}

func (s *Service) generateDescription(ctx context.Context, p describerepo.ProductRow) (string, error) {
	prompt := buildPrompt(p)

	text, err := s.llm.Complete(ctx, openai.CompleteOptions{
		Model: s.model,
		Messages: []openai.Message{
			{
				Role:    "system",
				Content: "Ты помощник интернет-магазина электрики и светотехники IQ-Home. Пиши краткие, точные описания товаров на русском языке. Описание должно быть 2-3 предложения. Не придумывай характеристики — опирайся только на название и категорию товара. Не используй маркетинговые клише.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 200,
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func buildPrompt(p describerepo.ProductRow) string {
	parts := []string{fmt.Sprintf("Товар: %s", p.NameRaw)}
	if p.Article != "" {
		parts = append(parts, fmt.Sprintf("Артикул: %s", p.Article))
	}
	if p.ProductType != "" {
		parts = append(parts, fmt.Sprintf("Категория: %s", p.ProductType))
	}
	if p.Brand != "" {
		parts = append(parts, fmt.Sprintf("Бренд: %s", p.Brand))
	}
	if p.Series != "" {
		parts = append(parts, fmt.Sprintf("Серия: %s", p.Series))
	}
	if p.Color != "" {
		parts = append(parts, fmt.Sprintf("Цвет: %s", p.Color))
	}
	parts = append(parts, "\nНапиши описание товара.")
	return strings.Join(parts, "\n")
}
