// Package compatibility analyses a user's cart and reports JASMART product
// incompatibilities using an LLM with a domain-specific system prompt.
package compatibility

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"iq-home/backend/internal/domain/compatibility"
)

// ─── interfaces ──────────────────────────────────────────────────────────────

type cartRepo interface {
	GetCartForCompatibility(ctx context.Context, userID string) ([]compatibility.CartItem, error)
}

type llm interface {
	Complete(ctx context.Context, opts CompleteOptions) (string, error)
}

// CompleteOptions mirrors chatsvc.CompleteOptions to avoid a circular import.
type CompleteOptions struct {
	Model    string
	Messages []Message
	JSONMode bool
}

type Message struct {
	Role    string
	Content any
}

// ─── Service ─────────────────────────────────────────────────────────────────

type Service struct {
	repo  cartRepo
	llm   llm
	model string
}

func New(repo cartRepo, llm llm, model string) *Service {
	return &Service{repo: repo, llm: llm, model: model}
}

// CheckCart loads the user's cart and asks the LLM for compatibility issues.
// Returns a Result with Compatible=true and empty Issues when nothing is wrong.
func (s *Service) CheckCart(ctx context.Context, userID string) (*compatibility.Result, error) {
	items, err := s.repo.GetCartForCompatibility(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("compatibility: load cart: %w", err)
	}

	if len(items) == 0 {
		return &compatibility.Result{Compatible: true, ItemCount: 0}, nil
	}

	answer, err := s.llm.Complete(ctx, CompleteOptions{
		Model:    s.model,
		JSONMode: true,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildCartMessage(items)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("compatibility: llm: %w", err)
	}

	result, err := parseResult(answer, len(items))
	if err != nil {
		return nil, fmt.Errorf("compatibility: parse: %w", err)
	}
	return result, nil
}

// ─── prompt ──────────────────────────────────────────────────────────────────

const systemPrompt = `Ты — эксперт по электрофурнитуре JASMART (Казахстан).
Твоя задача — проверить совместимость товаров в корзине покупателя и вернуть JSON.

## Правила совместимости JASMART

### 1. Рамки и механизмы — одна серия
Рамки, выключатели, розетки, диммеры должны быть одной серии: FD, G-серия или G-Flex.
Смешивать серии нельзя — рамка FD не подходит к механизму G-серии физически.

### 2. Диммеры и нагрузка
- Диммеры серии JASMART рассчитаны на резистивную нагрузку (лампы накаливания, галогенки).
- LED-лампы без нейтрального провода (2-проводная схема) несовместимы с большинством диммеров.
- Если в корзине есть диммер — предупреди, что нужно уточнить тип проводки и тип ламп.

### 3. Количество постов и рамка
- Рамка на 1 пост подходит только для 1 механизма.
- Рамка на 2 поста — для 2 механизмов, и т.д.
- Если количество механизмов (выключателей/розеток) не совпадает с суммой постов рамок — это проблема.

### 4. Проходные выключатели
- Проходной выключатель (схема управления из двух мест) требует минимум 2 штуки.
- Один проходной выключатель в корзине — предупреждение.

### 5. Розетки с заземлением
- Розетка SCHUKO (с заземлением) требует трёхпроводной разводки.
- Если в корзине есть SCHUKO-розетка — напомни об этом.

## Формат ответа (строго JSON, без markdown)
{
  "compatible": true | false,
  "issues": [
    {
      "type": "incompatible" | "warning",
      "product_ids": [id1, id2],
      "message": "Краткое описание проблемы на русском языке (1–2 предложения)."
    }
  ]
}

- "compatible": false если есть хотя бы одна проблема типа "incompatible".
- "issues": пустой массив [] если нет ни проблем, ни предупреждений.
- product_ids: ID товаров, которых касается проблема.
- Если корзина полностью совместима — вернуть {"compatible": true, "issues": []}.`

func buildCartMessage(items []compatibility.CartItem) string {
	var sb strings.Builder
	sb.WriteString("Проверь совместимость следующих товаров в корзине:\n\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf(
			"- ID %d | %q | Тип: %s | Configurator: %s | Серия: %s | Кол-во: %d\n",
			it.ProductID, it.Name,
			it.ProductType, it.ConfiguratorType,
			it.SeriesName, it.Quantity,
		))
	}
	return sb.String()
}

// ─── parser ──────────────────────────────────────────────────────────────────

func parseResult(raw string, itemCount int) (*compatibility.Result, error) {
	var parsed struct {
		Compatible bool `json:"compatible"`
		Issues     []struct {
			Type       string  `json:"type"`
			ProductIDs []int64 `json:"product_ids"`
			Message    string  `json:"message"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w (raw: %.200s)", err, raw)
	}

	result := &compatibility.Result{
		Compatible: parsed.Compatible,
		ItemCount:  itemCount,
	}
	for _, iss := range parsed.Issues {
		result.Issues = append(result.Issues, compatibility.Issue{
			Type:       iss.Type,
			ProductIDs: iss.ProductIDs,
			Message:    iss.Message,
		})
	}
	return result, nil
}
