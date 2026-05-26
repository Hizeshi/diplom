package chatsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/domain/product"
)

// ─── Dependency interfaces ────────────────────────────────────────────────────

type chatRepo interface {
	EnsureSession(ctx context.Context, sessionID, userID, platform string, authUserID *string, tokenHash string, trusted bool) (bool, error)
	GetSession(ctx context.Context, sessionID string) (*chat.Session, error)
	GetHistory(ctx context.Context, sessionID string, limit int) ([]chat.Message, error)
	GetPublicHistory(ctx context.Context, sessionID, tokenHash string, authUserID *string) ([]chat.PublicMessage, error)
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
	// 1. Determine token hash for session ownership.
	//    Trusted callers (internal API, Telegram) skip token enforcement.
	var rawToken string
	var tokenHash string
	if !req.Trusted {
		if req.SessionToken != "" {
			tokenHash = hashToken(req.SessionToken)
		} else {
			// First message — generate a new ownership token for this session.
			rawToken = generateToken()
			tokenHash = hashToken(rawToken)
		}
	}

	platform := req.Platform
	if platform == "" {
		platform = "web"
	}

	// 2. Ensure session exists and verify ownership.
	isNew, err := s.repo.EnsureSession(ctx, req.SessionID, req.UserID, platform, req.AuthUserID, tokenHash, req.Trusted)
	if err != nil {
		return nil, fmt.Errorf("chat: ensure session: %w", err)
	}
	// Only return rawToken when we actually created the session with it.
	if !isNew {
		rawToken = ""
	}

	// 3. Check human mode — if active, just save the message and return empty answer.
	session, err := s.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("chat: get session: %w", err)
	}
	if session != nil && session.IsHumanMode {
		_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", nil, "")
		return &chat.ChatResponse{Answer: "", SessionToken: rawToken}, nil
	}

	// 4. Short-circuit for ping.
	if isPing(req.Message) {
		return &chat.ChatResponse{Answer: "Привет! Чем могу помочь?", SessionToken: rawToken}, nil
	}

	// 5. Load conversation history.
	history, err := s.repo.GetHistory(ctx, req.SessionID, 20)
	if err != nil {
		s.log.Warn("chat: load history failed", "err", err)
	}

	// 6. Generate embedding for search.
	embedding, err := s.embedder.Embed(ctx, req.Message)
	if err != nil {
		s.log.Warn("chat: embed failed, continuing without vector", "err", err)
	}

	// 7. Decide whether to search products.
	matchCount := req.MatchCount
	if matchCount <= 0 {
		matchCount = 5
	}

	var (
		productMatches []chat.ProductMatch
		knowledgeItems []chat.KnowledgeMatch
	)

	if shouldSearchProducts(req.Message) {
		productMatches, knowledgeItems, err = s.runSearch(ctx, req.Message, embedding, matchCount, req.TopicFilter)
		if err != nil {
			s.log.Warn("chat: search failed", "err", err)
		}
	} else {
		// Always search knowledge base regardless of product search decision.
		knowledgeItems, _ = s.repo.SearchKnowledge(ctx, embedding, 0.65, 3, req.TopicFilter)
	}

	// 8. Build prompt and generate response.
	answer, err := s.generateResponse(ctx, req.Message, history, productMatches, knowledgeItems)
	if err != nil {
		return nil, fmt.Errorf("chat: generate: %w", err)
	}

	// 9. Save user message and assistant response.
	meta := map[string]any{"products_found": len(productMatches)}
	_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", nil, "")
	_ = s.repo.SaveMessage(ctx, req.SessionID, "assistant", answer, "assistant", "text", meta, "")

	return &chat.ChatResponse{
		Answer:       answer,
		Products:     productMatches,
		SessionToken: rawToken,
	}, nil
}

// GetPublicHistory returns the chat history for the session identified by sessionID.
// Access requires a valid session_token (anonymous sessions) or matching Supabase auth user.
func (s *Service) GetPublicHistory(ctx context.Context, sessionID, token string, authUserID *string) ([]chat.PublicMessage, error) {
	var tokenHash string
	if token != "" {
		tokenHash = hashToken(token)
	}
	msgs, err := s.repo.GetPublicHistory(ctx, sessionID, tokenHash, authUserID)
	if errors.Is(err, chat.ErrSessionForbidden) {
		return nil, chat.ErrSessionForbidden
	}
	return msgs, err
}

// ─── token helpers ────────────────────────────────────────────────────────────

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("chat: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
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
