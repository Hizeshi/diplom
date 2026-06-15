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
	"time"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/domain/product"
	"iq-home/backend/internal/domain/quote"
)

// ─── Dependency interfaces ────────────────────────────────────────────────────

type chatRepo interface {
	EnsureSession(ctx context.Context, sessionID, userID, platform string, authUserID *string, tokenHash string, trusted bool) (bool, error)
	GetSession(ctx context.Context, sessionID string) (*chat.Session, error)
	GetHistory(ctx context.Context, sessionID string, limit int) ([]chat.Message, error)
	GetPublicHistory(ctx context.Context, sessionID, tokenHash string, authUserID *string) ([]chat.PublicMessage, error)
	SaveMessage(ctx context.Context, sessionID, role, content, senderType, msgType string, meta map[string]any, filePath string) error
	SearchKnowledge(ctx context.Context, embedding []float32, threshold float64, limit int, topicFilter string) ([]chat.KnowledgeMatch, error)
	SetHumanMode(ctx context.Context, sessionID string, enabled bool, until *time.Time) error
	TrackOffTopic(ctx context.Context, sessionID string, reset bool) (int, error)
}

type productSearcher interface {
	Search(ctx context.Context, params product.SearchParams, embedding []float32) (*product.SearchResult, error)
	GetByIDs(ctx context.Context, ids []int64) ([]chat.ProductMatch, error)
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

type userContextRepo interface {
	GetContextHistory(ctx context.Context, userID string, limit int) ([]chat.ContextProduct, error)
	GetContextFavorites(ctx context.Context, userID string, limit int) ([]chat.ContextProduct, error)
	GetPopularProducts(ctx context.Context, limit int) ([]chat.ContextProduct, error)
}

type quoteGenerator interface {
	Generate(ctx context.Context, req quote.Request) (*quote.Response, error)
}

// ─── Service ─────────────────────────────────────────────────────────────────

type Service struct {
	repo      chatRepo
	products  productSearcher
	userCtx   userContextRepo
	quotes    quoteGenerator
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
	userCtx userContextRepo,
	quotes quoteGenerator,
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
		userCtx:  userCtx,
		quotes:   quotes,
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
		// Auto-disable if the cooldown period has passed.
		if session.AutoHumanUntil != nil && time.Now().After(*session.AutoHumanUntil) {
			_ = s.repo.SetHumanMode(ctx, req.SessionID, false, nil)
			_, _ = s.repo.TrackOffTopic(ctx, req.SessionID, true)
		} else {
			_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", nil, "")
			return &chat.ChatResponse{Answer: "", SessionToken: rawToken}, nil
		}
	}

	// 3b. Off-topic detection — block and track consecutive attempts.
	if isOffTopic(req.Message) {
		count, _ := s.repo.TrackOffTopic(ctx, req.SessionID, false)
		if count >= 3 {
			until := time.Now().Add(10 * time.Minute)
			_ = s.repo.SetHumanMode(ctx, req.SessionID, true, &until)
		}
		userMeta := req.MetaData
		if userMeta == nil {
			userMeta = map[string]any{}
		}
		_ = s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", "text", userMeta, "")
		_ = s.repo.SaveMessage(ctx, req.SessionID, "assistant", offTopicReply, "assistant", "text", map[string]any{}, "")
		return &chat.ChatResponse{Answer: offTopicReply, SessionToken: rawToken}, nil
	}
	// On-topic — reset consecutive off-topic counter.
	_, _ = s.repo.TrackOffTopic(ctx, req.SessionID, true)

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

	// 8. When КП is requested, recover the products discussed in recent history
	//    and prepend them so the prompt contains the exact items the user wants.
	if isKPRequest(req.Message) && len(productMatches) < matchCount {
		if histIDs := extractProductIDsFromHistory(history); len(histIDs) > 0 {
			histProducts, err := s.products.GetByIDs(ctx, histIDs)
			if err != nil {
				s.log.Warn("chat: get history products failed", "err", err)
			} else {
				// Merge: history products first, then de-duplicate with current search results.
				seen := make(map[int64]bool, len(histProducts))
				for _, p := range histProducts {
					seen[p.ID] = true
				}
				for _, p := range productMatches {
					if !seen[p.ID] {
						histProducts = append(histProducts, p)
						seen[p.ID] = true
					}
				}
				productMatches = histProducts
			}
		}
	}

	// 8b. Load user context for KP enrichment when the message is KP-related.
	var userCtxData chat.UserContext
	if shouldEnrichWithUserContext(req.Message) {
		if req.AuthUserID != nil {
			if h, err := s.userCtx.GetContextHistory(ctx, *req.AuthUserID, 8); err == nil {
				userCtxData.History = h
			}
			if f, err := s.userCtx.GetContextFavorites(ctx, *req.AuthUserID, 8); err == nil {
				userCtxData.Favorites = f
			}
		}
		if p, err := s.userCtx.GetPopularProducts(ctx, 8); err == nil {
			userCtxData.Popular = p
		}
	}

	// 9. Build prompt and generate response.
	answer, err := s.generateResponse(ctx, req.Message, history, productMatches, knowledgeItems, userCtxData)
	if err != nil {
		return nil, fmt.Errorf("chat: generate: %w", err)
	}

	// 9b. If this is a КП request with known products, auto-generate the PDF.
	var quoteURL string
	if isKPRequest(req.Message) && len(productMatches) > 0 {
		qty := extractQuantity(req.Message)
		items := make([]quote.Item, 0, len(productMatches))
		for _, p := range productMatches {
			article, _ := p.Metadata["article"].(string)
			items = append(items, quote.Item{
				Name:     p.Name,
				Article:  article,
				Quantity: qty,
				Price:    p.Price,
			})
		}
		if resp, qerr := s.quotes.Generate(ctx, quote.Request{Items: items}); qerr != nil {
			s.log.Warn("chat: auto-generate quote failed", "err", qerr)
		} else {
			quoteURL = resp.URL
		}
	}

	// 10. Save user message and assistant response.
	msgType := req.MessageType
	if msgType == "" {
		msgType = "text"
	}
	userMeta := req.MetaData
	if userMeta == nil {
		userMeta = map[string]any{}
	}
	if err := s.repo.SaveMessage(ctx, req.SessionID, "user", req.Message, "user", msgType, userMeta, req.FilePath); err != nil {
		s.log.Warn("chat: save user message failed", "session", req.SessionID, "err", err)
	}
	assistantMeta := map[string]any{"products_found": len(productMatches)}
	if len(productMatches) > 0 {
		ids := make([]int64, len(productMatches))
		for i, p := range productMatches {
			ids[i] = p.ID
		}
		assistantMeta["product_ids"] = ids
	}
	if quoteURL != "" {
		assistantMeta["quote_url"] = quoteURL
	}
	if err := s.repo.SaveMessage(ctx, req.SessionID, "assistant", answer, "assistant", "text", assistantMeta, ""); err != nil {
		s.log.Warn("chat: save assistant message failed", "session", req.SessionID, "err", err)
	}

	return &chat.ChatResponse{
		Answer:       answer,
		Products:     productMatches,
		QuoteURL:     quoteURL,
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
