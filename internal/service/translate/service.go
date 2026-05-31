package translatesvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"iq-home/backend/internal/client/openai"
)

const batchSize = 15

type repo interface {
	FetchUntranslated(ctx context.Context, locale string, limit int) ([]ProductRow, error)
	SaveTranslations(ctx context.Context, locale string, items []TranslatedItem) error
	CountUntranslated(ctx context.Context, locale string) (int, error)
}

type ProductRow struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      string `json:"params"`
}

type TranslatedItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Params      string `json:"params"`
}

type Result struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

type Service struct {
	repo   repo
	openai *openai.Client
	model  string
}

func New(repo repo, openai *openai.Client, model string) *Service {
	return &Service{repo: repo, openai: openai, model: model}
}

func (s *Service) Translate(ctx context.Context, locale string) (*Result, error) {
	total, err := s.repo.CountUntranslated(ctx, locale)
	if err != nil {
		return nil, fmt.Errorf("translate: count: %w", err)
	}

	result := &Result{Total: total}

	for {
		rows, err := s.repo.FetchUntranslated(ctx, locale, batchSize)
		if err != nil {
			return result, fmt.Errorf("translate: fetch batch: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		translated, err := s.translateBatch(ctx, rows, locale)
		if err != nil {
			slog.Error("translate batch failed", "locale", locale, "count", len(rows), "err", err)
			result.Failed += len(rows)
			// Mark them as attempted with empty string so we skip next time
			// and don't loop forever on broken products.
			placeholders := make([]TranslatedItem, len(rows))
			for i, r := range rows {
				placeholders[i] = TranslatedItem{ID: r.ID, Name: r.Name}
			}
			_ = s.repo.SaveTranslations(ctx, locale, placeholders)
			continue
		}

		if err := s.repo.SaveTranslations(ctx, locale, translated); err != nil {
			slog.Error("save translations failed", "locale", locale, "err", err)
			result.Failed += len(rows)
		} else {
			result.Processed += len(translated)
		}

		if len(rows) < batchSize {
			break
		}
	}

	return result, nil
}

func (s *Service) translateBatch(ctx context.Context, rows []ProductRow, locale string) ([]TranslatedItem, error) {
	langName := map[string]string{
		"kk": "Kazakh",
		"en": "English",
	}[locale]
	if langName == "" {
		return nil, fmt.Errorf("unsupported locale: %s", locale)
	}

	inputJSON, _ := json.Marshal(rows)

	systemPrompt := fmt.Sprintf(`You are a professional translator specializing in electrical engineering and smart home products (JASMART brand).
Translate the product data from Russian to %s.
Rules:
- Translate name, description, and params fields
- Keep technical terms accurate (switch, dimmer, socket, frame, series names like FD/G-Flex)
- If description or params is empty string "", keep it as ""
- Do NOT translate brand names (JASMART, SCHUKO) or article codes
- Return ONLY a JSON array: [{"id":N,"name":"...","description":"...","params":"..."}]`, langName)

	resp, err := s.openai.Complete(ctx, openai.CompleteOptions{
		Model:    s.model,
		JSONMode: true,
		Messages: []openai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(inputJSON)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}

	var items []TranslatedItem
	if err := json.Unmarshal([]byte(resp), &items); err != nil {
		// Try wrapped {"items":[...]} format
		var wrapped struct {
			Items []TranslatedItem `json:"items"`
		}
		if err2 := json.Unmarshal([]byte(resp), &wrapped); err2 != nil {
			return nil, fmt.Errorf("parse response: %w (raw: %.300s)", err, resp)
		}
		items = wrapped.Items
	}

	return items, nil
}
