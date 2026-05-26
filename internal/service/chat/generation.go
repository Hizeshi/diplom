package chatsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"iq-home/backend/internal/domain/chat"
)

// CompleteOptions mirrors openai.CompleteOptions to avoid importing client package.
type CompleteOptions struct {
	Model    string
	Messages []LLMMessage
	JSONMode bool
}

type LLMMessage struct {
	Role    string
	Content any // string | []ContentPart
}

type ContentPart struct {
	Type     string
	Text     string
	ImageURL string
}

// shouldSearchProducts asks the LLM whether the user's message requires a product search.
func (s *Service) shouldSearchProducts(ctx context.Context, message string, history []chat.Message) (bool, error) {
	systemPrompt := `Ты определяешь, нужен ли поиск товаров для ответа на сообщение пользователя.
Отвечай только JSON: {"search": true} или {"search": false}.
Поиск нужен если пользователь спрашивает о конкретных товарах, характеристиках, ценах, наличии, сравнивает товары.
Поиск НЕ нужен для приветствий, вопросов о доставке/оплате/гарантии, жалоб, общих вопросов.`

	msgs := []LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: message},
	}

	resp, err := s.llm.Complete(ctx, CompleteOptions{
		Model:    s.cfg.OpenAIModel,
		Messages: msgs,
		JSONMode: true,
	})
	if err != nil {
		return true, err // default to search on error
	}

	var result struct {
		Search bool `json:"search"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return true, nil
	}
	return result.Search, nil
}

// generateResponse builds the full prompt and calls the LLM.
func (s *Service) generateResponse(
	ctx context.Context,
	userMessage string,
	history []chat.Message,
	products []chat.ProductMatch,
	knowledge []chat.KnowledgeMatch,
) (string, error) {
	systemPrompt := s.buildSystemPrompt(products, knowledge)

	msgs := []LLMMessage{{Role: "system", Content: systemPrompt}}

	// Add conversation history.
	for _, h := range history {
		msgs = append(msgs, LLMMessage{Role: h.Role, Content: h.Content})
	}

	msgs = append(msgs, LLMMessage{Role: "user", Content: userMessage})

	answer, err := s.llm.Complete(ctx, CompleteOptions{
		Model:    s.cfg.OpenAIModel,
		Messages: msgs,
	})
	if err != nil {
		return "", fmt.Errorf("generation: complete: %w", err)
	}
	return answer, nil
}

func (s *Service) buildSystemPrompt(products []chat.ProductMatch, knowledge []chat.KnowledgeMatch) string {
	var sb strings.Builder

	sb.WriteString(`Ты — умный AI-ассистент интернет-магазина IQ Home, специализирующегося на напольных покрытиях, `)
	sb.WriteString(`плинтусах и сопутствующих товарах. Помогай клиентам подбирать товары, отвечай на вопросы о характеристиках, `)
	sb.WriteString(`ценах, наличии. Будь вежливым, профессиональным и конкретным. Отвечай на русском языке.`)
	sb.WriteString("\n\n")

	if len(products) > 0 {
		sb.WriteString("## Найденные товары:\n")
		for _, p := range products {
			sb.WriteString(fmt.Sprintf("- **%s** — %.0f тг", p.Name, p.Price))
			if brand, ok := p.Metadata["brand"].(string); ok && brand != "" {
				sb.WriteString(fmt.Sprintf(", бренд: %s", brand))
			}
			if color, ok := p.Metadata["color"].(string); ok && color != "" {
				sb.WriteString(fmt.Sprintf(", цвет: %s", color))
			}
			if s.cfg.ProductURL != "" {
				sb.WriteString(fmt.Sprintf(" — [ссылка](%s%d)", s.cfg.ProductURL, p.ID))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(knowledge) > 0 {
		sb.WriteString("## Справочная информация:\n")
		for _, k := range knowledge {
			sb.WriteString(k.Content)
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString("Если товар не найден — предложи уточнить запрос. Не выдумывай товары и цены.")

	return sb.String()
}

// analyzeImage sends an image to the vision model and returns a search-ready description.
func (s *Service) analyzeImage(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	// Encode image as base64 data URL.
	encoded := encodeBase64(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	msgs := []LLMMessage{
		{
			Role: "user",
			Content: []ContentPart{
				{
					Type: "image_url",
					ImageURL: dataURL,
				},
				{
					Type: "text",
					Text: "Определи тип напольного покрытия на изображении: материал, цвет, декор/паттерн, бренд если виден. Дай краткое описание для поиска похожих товаров.",
				},
			},
		},
	}

	return s.llm.Complete(ctx, CompleteOptions{
		Model:    s.cfg.OpenAIVisionModel,
		Messages: msgs,
	})
}

func encodeBase64(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var sb strings.Builder
	for i := 0; i < len(data); i += 3 {
		b := [3]byte{}
		n := copy(b[:], data[i:])
		sb.WriteByte(chars[b[0]>>2])
		sb.WriteByte(chars[(b[0]&0x3)<<4|b[1]>>4])
		if n > 1 {
			sb.WriteByte(chars[(b[1]&0xF)<<2|b[2]>>6])
		} else {
			sb.WriteByte('=')
		}
		if n > 2 {
			sb.WriteByte(chars[b[2]&0x3F])
		} else {
			sb.WriteByte('=')
		}
	}
	return sb.String()
}
