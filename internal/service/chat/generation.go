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
// shouldSearchProducts decides locally — without an LLM call — whether the message
// requires a product search. Tuned for an electrical accessories store (sockets,
// switches, frames, dimmers, etc.).
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
		// типы товаров
		"розетк", "выключател", "рамк", "диммер", "светорегулятор", "заглушк",
		"кнопк", "механизм", "модул", "вывод кабел", "tv", "fm", "телевизионн",
		"компьютерн", "телефонн", "проходн",
		// характеристики
		"ампер", "вольт", "10a", "16a", "250v", "одноклавишн", "двухклавишн",
		"трёхклавишн", "одноместн", "двухместн", "постов",
		// серии и бренды
		"jasmart", "fd-серия", "g-серия", "g-flex", "серия", "коллекция",
		// цвет / материал
		"белый", "белая", "чёрный", "бежев", "серый", "золот", "хром",
		"бронза", "мокко", "тауп", "матов", "глянц",
		// покупка
		"цена", "цены", "стоимост", "сколько стоит", "почём", "прайс",
		"купить", "заказать", "приобрести", "хочу", "нужен", "нужна", "нужно",
		"подобрать", "выбрать", "посоветуй", "порекоменд", "какой", "какая", "какие",
		"наличи", "есть ли", "в наличии",
		"артикул", "модел", "сравни", "отличи", "разниц", "лучше",
		"акция", "скидк", "комплект",
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

// extractQuantity pulls the first number from КП messages like "на 10 штук", "5 шт".
// Returns 1 if no number found.
func extractQuantity(message string) int {
	msg := strings.ToLower(message)
	// Look for digits preceded/followed by quantity words.
	patterns := []string{
		"на ", " шт", " штук", " единиц", " комплект",
	}
	for _, p := range patterns {
		idx := strings.Index(msg, p)
		if idx < 0 {
			continue
		}
		// Scan backwards from pattern for digits.
		start := idx - 1
		for start >= 0 && msg[start] == ' ' {
			start--
		}
		end := start + 1
		for start >= 0 && msg[start] >= '0' && msg[start] <= '9' {
			start--
		}
		if end > start+1 {
			n := 0
			for _, c := range msg[start+1 : end] {
				n = n*10 + int(c-'0')
			}
			if n > 0 && n <= 10000 {
				return n
			}
		}
	}
	// Fallback: find any standalone number in the message.
	inDigit := false
	num := 0
	for _, c := range msg {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
			inDigit = true
		} else {
			if inDigit && num > 0 && num <= 10000 {
				return num
			}
			inDigit = false
			num = 0
		}
	}
	return 1
}

// isKPRequest detects explicit КП / quote assembly intent.
func isKPRequest(message string) bool {
	msg := strings.ToLower(message)
	keywords := []string{
		"собери кп", "составь кп", "сделай кп", "кп на",
		"коммерческое предложение", "собери предложение",
		"возьму", "беру", "оформи", "закажи", "собери комплект",
		"да, собери", "да собери", "давай кп", "хочу кп",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// wantsKP reports whether the user wants a КП assembled — either by explicit
// request, or by confirming a КП offer the assistant just made (so a bare
// "да" / "давай" / "собери" in reply to "могу оформить КП?" works).
func wantsKP(message string, history []chat.Message) bool {
	if isKPRequest(message) {
		return true
	}
	return assistantOfferedKP(history) && isAffirmative(message)
}

// isAffirmative detects a short confirmation reply. Length-bounded so a new
// question that merely starts with "да" is not mistaken for a confirmation.
func isAffirmative(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	msg = strings.Trim(msg, " .!,)?–-\"")
	if msg == "" || len([]rune(msg)) > 25 {
		return false
	}
	affirmatives := []string{
		"да", "ага", "угу", "конечно", "давай", "давайте", "ок", "окей",
		"хорошо", "согласен", "согласна", "да давай", "да конечно", "да хочу",
		"да, хочу", "да, давай", "да, конечно", "хочу", "можно", "сделай",
		"сделайте", "соберите", "собери", "оформите", "да пожалуйста",
		"давай да", "ну давай", "валяй", "yes", "ok", "окей давай",
	}
	for _, a := range affirmatives {
		if msg == a {
			return true
		}
	}
	return false
}

// assistantOfferedKP reports whether the most recent assistant message offered
// to assemble a КП, so a bare affirmative can be read as confirmation.
func assistantOfferedKP(history []chat.Message) bool {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role != "assistant" {
			continue
		}
		msg := strings.ToLower(history[i].Content)
		if !strings.Contains(msg, "кп") && !strings.Contains(msg, "коммерческое предложение") {
			return false
		}
		offerVerbs := []string{
			"могу", "хотите", "хотели", "оформ", "собр", "сформир",
			"подготов", "составл", "составить", "предлож", "сделать",
		}
		for _, v := range offerVerbs {
			if strings.Contains(msg, v) {
				return true
			}
		}
		return false // only the latest assistant message matters
	}
	return false
}

// extractProductIDsFromHistory collects product_ids saved in the last 5
// assistant messages so we can re-fetch them when building a КП.
func extractProductIDsFromHistory(history []chat.Message) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	checked := 0
	for i := len(history) - 1; i >= 0 && checked < 5; i-- {
		msg := history[i]
		if msg.Role != "assistant" || msg.MetaData == nil {
			continue
		}
		checked++
		raw, ok := msg.MetaData["product_ids"]
		if !ok {
			continue
		}
		// MetaData is decoded from JSON, so numbers come as float64.
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if f, ok := item.(float64); ok {
					id := int64(f)
					if !seen[id] {
						seen[id] = true
						ids = append(ids, id)
					}
				}
			}
		case []int64:
			for _, id := range v {
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}

// isOffTopic returns true when the message is clearly unrelated to the store.
// It first checks for strong store signals (always on-topic) and then for
// off-topic signals to avoid false positives on short/ambiguous messages.
func isOffTopic(message string) bool {
	msg := strings.ToLower(message)

	// Always on-topic: anything that mentions store-relevant terms.
	storeSignals := []string{
		"розетк", "выключател", "рамк", "диммер", "заглушк", "механизм", "jasmart",
		"серия", "коллекци", "артикул", "модел", "цена", "цены", "тенге", "тг",
		"купить", "заказ", "наличи", "доставк", "магазин", "товар", "продукт",
		"кп", "коммерческ", "предложени", "электро", "230v", "10a", "16a",
		"iq home", "iq-home", "l-xor",
	}
	for _, kw := range storeSignals {
		if strings.Contains(msg, kw) {
			return false
		}
	}

	// Strong off-topic signals.
	offTopicSignals := []string{
		// Программирование
		"python", "javascript", "java ", "golang", "c++", "c#", "php", "ruby", "kotlin", "swift",
		"код", "кодирован", "программирован", "алгоритм", "сортировк", "функци", "перемен",
		"массив", "цикл", "рекурс", "компилятор", "синтакс", "дебаг", "баг", "github",
		// Кулинария / быт
		"рецепт", "готовить", "блюд", "кухн", "еда", "суп", "борщ", "пицц", "торт",
		// Медицина
		"лечение", "болезн", "таблетк", "врач", "клиник", "симптом", "диагноз",
		// Развлечения
		"фильм", "сериал", "кино", "игр", "музык", "песн", "артист", "актёр",
		// Обучение / наука
		"математик", "физик", "химия", "биологи", "история", "география", "урок", "реферат",
		// Финансы / крипто
		"биткоин", "крипто", "акци", "инвестиц", "доллар", "валют", "forex",
		// Прочее
		"погода", "гороскоп", "анекдот", "шутк", "стихотворен", "поэзи",
		"политик", "новости", "президент", "война", "выбор",
	}
	for _, kw := range offTopicSignals {
		if strings.Contains(msg, kw) {
			return true
		}
	}

	return false
}

// offTopicReply is the standard refusal message sent to off-topic requests.
const offTopicReply = "Я специализируюсь только на вопросах, связанных с электрофурнитурой магазина IQ Home: розетки, выключатели, рамки, диммеры и сопутствующие товары бренда JASMART. Чем могу помочь по этой теме? 😊"

// shouldEnrichWithUserContext returns true when the message suggests the user
// wants a КП, recommendations from their history/favorites, or popular items.
func shouldEnrichWithUserContext(message string) bool {
	msg := strings.ToLower(message)
	keywords := []string{
		"кп", "коммерческ", "предложени",
		"избранн", "избранног", "сохранённ", "закладк",
		"смотрел", "просматривал", "историю", "из истории",
		"популярн", "хит", "бестселлер", "чаще всего",
		"посоветуй", "порекомендуй", "что взять", "подбери",
	}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
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
	userCtx chat.UserContext,
) (string, error) {
	ctx, span := tracer.Start(ctx, "rag.generate_llm")
	defer span.End()

	systemPrompt := s.buildSystemPrompt(products, knowledge, userCtx)

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

func (s *Service) buildSystemPrompt(products []chat.ProductMatch, knowledge []chat.KnowledgeMatch, userCtx chat.UserContext) string {
	var sb strings.Builder

	sb.WriteString("Ты — Алина, менеджер по продажам интернет-магазина IQ Home (Казахстан).\n")
	sb.WriteString("Магазин специализируется на электрофурнитуре бренда JASMART: розетки, выключатели, рамки, диммеры, заглушки и сопутствующие механизмы.\n\n")

	sb.WriteString("## Твои задачи:\n")
	sb.WriteString("- Помогать клиенту подобрать нужную фурнитуру: тип, серию, цвет, количество постов.\n")
	sb.WriteString("- Называть конкретные товары из списка ниже с ценами и ссылками.\n")
	sb.WriteString("- Предлагать комплектацию: к выключателю — рамку той же серии и цвета; к розетке — заглушку или рамку.\n")
	sb.WriteString("- Объяснять характеристики: номинал (10A/16A), тип (проходной, одноклавишный и т.д.), степень защиты.\n")
	sb.WriteString("- Уточнять потребности: сколько постов, какой цвет интерьера, нужна ли заземляющая розетка.\n")
	sb.WriteString("- Мягко предлагать оформить заказ или перейти к товару по ссылке.\n\n")

	sb.WriteString("## Правила:\n")
	sb.WriteString("- Отвечай только на русском языке, дружелюбно и профессионально.\n")
	sb.WriteString("- Используй только товары и цены из списка ниже — ничего не выдумывай.\n")
	sb.WriteString("- Если подходящего товара нет — скажи честно и предложи уточнить запрос или позвонить менеджеру.\n")
	sb.WriteString("- Ответы — 3–5 предложений, без лишних вступлений и воды.\n")
	sb.WriteString("- Цены указывай в тенге (тг).\n")
	sb.WriteString("- СТРОГО: ты отвечаешь ТОЛЬКО на вопросы об электрофурнитуре магазина IQ Home. На любые другие темы (программирование, рецепты, медицина, развлечения, наука, политика и т.д.) вежливо отказывай и перенаправляй к теме магазина.\n\n")

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

	if len(userCtx.History) > 0 {
		sb.WriteString("## Товары, которые пользователь недавно смотрел:\n")
		for i, p := range userCtx.History {
			fmt.Fprintf(&sb, "%d. %s — %.0f тг\n", i+1, p.Name, p.Price)
		}
		sb.WriteString("Можешь включить их в КП, если они подходят по теме.\n\n")
	}

	if len(userCtx.Favorites) > 0 {
		sb.WriteString("## Избранные товары пользователя:\n")
		for i, p := range userCtx.Favorites {
			fmt.Fprintf(&sb, "%d. %s — %.0f тг\n", i+1, p.Name, p.Price)
		}
		sb.WriteString("Эти товары пользователь сохранил — приоритетно включи их в КП.\n\n")
	}

	if len(userCtx.Popular) > 0 {
		sb.WriteString("## Популярные товары магазина (часто заказывают):\n")
		for i, p := range userCtx.Popular {
			fmt.Fprintf(&sb, "%d. %s — %.0f тг\n", i+1, p.Name, p.Price)
		}
		sb.WriteString("Можешь предложить их в дополнение к КП.\n\n")
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
