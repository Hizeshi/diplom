package chat

import (
	"errors"
	"time"
)

var ErrSessionForbidden = errors.New("chat: session access denied")

// ─── Session ─────────────────────────────────────────────────────────────────

type Session struct {
	SessionID      string
	AuthUserID     *string
	UserID         string
	Platform       string
	IsHumanMode    bool
	AutoHumanUntil *time.Time // non-nil = human mode was auto-enabled; reset when past
	OffTopicCount  int        // consecutive off-topic messages
}

// ─── Messages ────────────────────────────────────────────────────────────────

type Message struct {
	Role        string
	Content     string
	SenderType  string
	MessageType string
	MetaData    map[string]any
	FilePath    string
	CreatedAt   time.Time
}

type PublicMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	SenderType string `json:"sender_type"`
	Time       string `json:"time"` // "HH:MM"
}

// ─── Request / Response ──────────────────────────────────────────────────────

type ChatRequest struct {
	Message      string
	SessionID    string
	SessionToken string  // anonymous ownership token; empty for internal/telegram callers
	AuthUserID   *string // Supabase UUID (site users)
	UserID       string  // arbitrary string (telegram chat id, etc.)
	MatchCount   int
	TopicFilter  string
	Platform     string // "web" | "telegram" | "internal"
	Trusted      bool   // true for /v1 internal and telegram routes

	// Media fields — set by ProcessMedia so HandleMessage saves the right type.
	MessageType string         // "text" (default) | "voice" | "photo" | "document"
	FilePath    string         // storage path of uploaded attachment
	MetaData    map[string]any // extra metadata stored alongside the user message
}

type ChatResponse struct {
	Answer       string         `json:"answer"`
	Products     []ProductMatch `json:"products,omitempty"`
	QuoteURL     string         `json:"kp_pdf_url,omitempty"`
	SessionToken string         `json:"session_token,omitempty"` // returned on new session creation
}

// ─── Search Results ──────────────────────────────────────────────────────────

type ProductMatch struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	Price      float64        `json:"price"`
	Score      float64        `json:"score"`
	ImageURL   string         `json:"image_url,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type KnowledgeMatch struct {
	ID         int64          `json:"id"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Similarity float64        `json:"similarity"`
}

// ─── User context for KP enrichment ─────────────────────────────────────────

// ContextProduct is a product from the user's history, favorites, or the
// global popular list. Used to enrich the KP prompt.
type ContextProduct struct {
	ID    int64
	Name  string
	Price float64
}

type UserContext struct {
	History   []ContextProduct
	Favorites []ContextProduct
	Popular   []ContextProduct
}

// ─── Media ───────────────────────────────────────────────────────────────────

type MediaRequest struct {
	SessionID   string
	AuthUserID  *string
	UserID      string
	Platform    string
	MessageType string // "voice" | "photo" | "document"
	Data        []byte
	Filename    string
	MimeType    string
	Trusted     bool
}
