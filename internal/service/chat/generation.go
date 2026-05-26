package chatsvc

import (
	"context"
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

// shouldSearchProducts decides locally — without an LLM call — whether the message
// requires a product search. This saves one OpenAI round-trip per message.
// The classifier uses keyword lists tuned for a flooring/home-finish store.
func shouldSearchProducts(message string) bool {
	msg := strings.ToLower(message)

	// Explicit non-product topics → skip search.
	noSearchKeywords := []string{
		"доставк", "оплат", "оплатить", "гарантия", "гарантийн", "возврат", "обмен",
		"реквизит", "договор", "контакт", "адрес", "телефон", "график", "режим",
		"спасибо", "благодарю", "отлично", "понял", "ок ", "окей",
	}
	for _, kw := range noSearchKeywords {
		if strings.Contains(msg, kw) {
			return false
		}
	}

	// Product-related signals → run search.
	searchKeywords := []string{
		"ламинат", "паркет", "линолеум", "vinyl", "винил", "плинтус", "подложк",
		"покрыти", "напольн", "укладк", "пол ", "полы", "полу", "полов",
		"цена", "цены", "стоимост", "сколько стоит", "почём", "прайс",
		"купить", "заказать", "приобрести", "хочу", "нужен", "нужна", "нужно",
		"подобрать", "выбрать", "посоветуй", "порекоменд", "какой", "какая", "какие",
		"наличи", "есть ли", "есть в наличии", "в наличии",
		"характеристик", "размер", "толщин", "ширин", "класс", "износостойк",
		"цвет", "оттенок", "декор", "текстур", "рисунок",
		"бренд", "производитель", "коллекция", "серия",
		"артикул", "модел",
		"сравни", "отличи", "разниц", "лучше",
		"акция", "скидк", "распродаж", "спецпредложени",
		"комплект", "монтаж", "установк",
	}
	for _, kw := range searchKeywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}

	// Short messages with a question mark are likely product queries.
	if strings.Contains(msg, "?") && len([]rune(msg)) < 80 {
		return true
	}

	return false
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

	sb.WriteString(`Ты — Алина, опытный менеджер по продажам интернет-магазина IQ Home (Казахстан).` + "\n")
	sb.WriteString(`Магазин специализируется на напольных покрытиях: ламинат, паркетная доска, LVT/SPC vinyl, линолеум, плинтусы и подложки.` + "\n\n")

	sb.WriteString("## Твои задачи:\n")
	sb.WriteString("- Помогать клиенту выбрать подходящее покрытие под его условия (тип помещения, нагрузка, бюджет, стиль).\n")
	sb.WriteString("- Называть конкретные товары из списка ниже с ценами и ссылками.\n")
	sb.WriteString("- Предлагать сопутствующие товары: к ламинату — подложку и плинтус; к паркету — масло/лак.\n")
	sb.WriteString("- Отвечать на вопросы о характеристиках, классах износостойкости, монтаже, уходе.\n")
	sb.WriteString("- Мягко подталкивать к покупке: уточнять площадь, сроки, предлагать оформить заказ.\n\n")

	sb.WriteString("## Правила:\n")
	sb.WriteString("- Отвечай только на русском языке, дружелюбно и профессионально.\n")
	sb.WriteString("- Не выдумывай товары, цены и характеристики — используй только данные из списка ниже.\n")
	sb.WriteString("- Если подходящего товара нет — честно скажи и предложи уточнить запрос или позвонить менеджеру.\n")
	sb.WriteString("- Ответы делай краткими (3–5 предложений), без лишних вступлений.\n")
	sb.WriteString("- Цены указывай в тенге (тг).\n\n")

	if len(products) > 0 {
		sb.WriteString("## Доступные товары по запросу клиента:\n")
		for i, p := range products {
			sb.WriteString(fmt.Sprintf("%d. **%s** — %.0f тг", i+1, p.Name, p.Price))
			if brand, ok := p.Metadata["brand"].(string); ok && brand != "" {
				sb.WriteString(fmt.Sprintf(" | Бренд: %s", brand))
			}
			if color, ok := p.Metadata["color"].(string); ok && color != "" {
				sb.WriteString(fmt.Sprintf(" | Цвет: %s", color))
			}
			if series, ok := p.Metadata["series"].(string); ok && series != "" {
				sb.WriteString(fmt.Sprintf(" | Серия: %s", series))
			}
			if s.cfg.ProductURL != "" {
				sb.WriteString(fmt.Sprintf(" | [Открыть товар](%s%d)", s.cfg.ProductURL, p.ID))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(knowledge) > 0 {
		sb.WriteString("## Справочная информация о магазине:\n")
		for _, k := range knowledge {
			sb.WriteString(k.Content)
			sb.WriteString("\n\n")
		}
	}

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
