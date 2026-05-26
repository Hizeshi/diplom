package chatsvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/domain/product"
)

// ─── Dependency interfaces ────────────────────────────────────────────────────

type chatRepo interface {
	EnsureSession(ctx context.Context, sessionID, userID, platform string, authUserID *string) error
	GetSession(ctx context.Context, sessionID string) (*chat.Session, error)
	GetHistory(ctx context.Context, sessionID string, limit int) ([]chat.Message, error)
	GetPublicHistory(ctx context.Context, sessionID string) ([]chat.PublicMessage, error)
	SaveMessage(ctx context.Context, sessionID, role, content, senderType, msgType string, meta map[string]any, filePath string) error
	SearchKnowledge(ctx context.Context, embedding []float32, threshold float64, limit int, topicFilter string) ([]chat.KnowledgeMatch, error)
}

type productSearcher interface {
	Search(ctx context.Context, params product.SearchParams, embedding []float32) (*product.SearchResult, error)
}

type embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type llm interface {
	Complete(ctx context.Context, opts CompleteOptions) (string, error)
	Transcribe(ctx context.Context, model string, data []byte, filename string) (string, error)
}

type docParser interface {
	ExtractText(ctx context.Context, data []byte, contentType string) (string, error)
}

type fileStore interface {
	Upload(ctx context.Context, bucket, path string, data []byte, contentType string) (string, error)
}

// ─── Service ─────────────────────────────────────────────────────────────────

type Service struct {
	repo      chatRepo
	products  productSearcher
	embedder  embedder
	llm       llm
	parser    docParser
	store     fileStore
	log       *slog.Logger
	cfg       Config
}

type Config struct {
	OpenAIModel         string
	OpenAIVisionModel   string
	OpenAITranscribeModel string
	ProductURL          string // base URL for product links, e.g. https://iq-home.kz/products/
}

func New(
	repo chatRepo,
	products productSearcher,
	embedder embedder,
	llm llm,
	parser docParser,
	store fileStore,
	log *slog.Logger,
	cfg Config,
) *Service {
	return &Service{
		repo:     repo,
		products: products,
		embedder: embedder,
		llm:      llm,
		parser:   parser,
		store:    store,
		log:      log,
		cfg:      cfg,
	}
}

// ─── HandleMessage ────────────────────────────────────────────────────────────

func (s *Service) HandleMessage(ctx context.Context, req chat.ChatRequest) (*chat.ChatResponse, error) {
	// 1. Ensure session exists.
	platform := req.Platform
	if platform == "" {
		platform = "web"
	}
	if err := s.repo.EnsureSession(ctx, req.SessionID, req.UserID, platform, req.AuthUserID); err != nil {
		return nil, fmt.Errorf("chat: ensure session: %w", err)
	}

	// 2. Check human mode — if active, just save the message and return empty answer.
	session, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("chat: get session: %w", err)
	}
	if session != nil && session.IsHumanMode {
		_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", nil, "")
		return &chat.ChatResponse{Answer: ""}, nil
	}

	// 3. Short-circuit for ping.
	if isPing(req.Message) {
		return &chat.ChatResponse{Answer: "Привет! Чем могу помочь?"}, nil
	}

	// 4. Load conversation history.
	history, err := s.repo.GetHistory(ctx, req.SessionID, 20)
	if err != nil {
		s.log.Warn("chat: load history failed", "err", err)
	}

	// 5. Generate embedding for search.
	embedding, err := s.embedder.Embed(ctx, req.Message)
	if err != nil {
		s.log.Warn("chat: embed failed, continuing without vector", "err", err)
	}

	// 6. Decide whether to search products.
	matchCount := req.MatchCount
	if matchCount <= 0 {
		matchCount = 5
	}

	var (
		productMatches  []chat.ProductMatch
		knowledgeItems  []chat.KnowledgeMatch
	)

	needsSearch, _ := s.shouldSearchProducts(ctx, req.Message, history)
	if needsSearch {
		productMatches, knowledgeItems, err = s.runSearch(ctx, req.Message, embedding, matchCount, req.TopicFilter)
		if err != nil {
			s.log.Warn("chat: search failed", "err", err)
		}
	} else {
		// Always search knowledge base regardless of product search decision.
		knowledgeItems, _ = s.repo.SearchKnowledge(ctx, embedding, 0.65, 3, req.TopicFilter)
	}

	// 7. Build prompt and generate response.
	answer, err := s.generateResponse(ctx, req.Message, history, productMatches, knowledgeItems)
	if err != nil {
		return nil, fmt.Errorf("chat: generate: %w", err)
	}

	// 8. Save user message and assistant response.
	meta := map[string]any{"products_found": len(productMatches)}
	_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", nil, "")
	_ = s.repo.SaveMessage(ctx, req.SessionID, "assistant", answer, "assistant", "text", meta, "")

	return &chat.ChatResponse{
		Answer:   answer,
		Products: productMatches,
	}, nil
}

// GetPublicHistory returns the chat history for public display.
func (s *Service) GetPublicHistory(ctx context.Context, sessionID string) ([]chat.PublicMessage, error) {
	return s.repo.GetPublicHistory(ctx, sessionID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func isPing(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	pings := []string{"ping", "привет", "здравствуйте", "добрый день", "добрый вечер", "hi", "hello"}
	for _, p := range pings {
		if msg == p {
			return true
		}
	}
	return false
}
