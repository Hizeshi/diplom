package chathandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/middleware"
	"iq-home/backend/pkg/respond"
	"iq-home/backend/pkg/validate"
)

type service interface {
	HandleMessage(ctx context.Context, req chat.ChatRequest) (*chat.ChatResponse, error)
	GetPublicHistory(ctx context.Context, sessionID, token string, authUserID *string) ([]chat.PublicMessage, error)
	ProcessMedia(ctx context.Context, req chat.MediaRequest) (*chat.ChatResponse, error)
}

type Handler struct {
	svc service
}

func New(svc service) *Handler {
	return &Handler{svc: svc}
}

// ─── POST /api/chat  (Supabase JWT optional) ─────────────────────────────────
// ─── POST /v1/chat   (X-Internal-Token) ──────────────────────────────────────

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message      string `json:"message"       validate:"required,min=1,max=2000"`
		SessionID    string `json:"session_id"    validate:"required,min=1,max=100"`
		SessionToken string `json:"session_token"`
		UserID       string `json:"user_id"`
		MatchCount   int    `json:"match_count"`
		TopicFilter  string `json:"topic_filter"`
		Platform     string `json:"platform"`
	}
	if !validate.DecodeAndValidate(w, r, &body) {
		return
	}
	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		respond.BadRequest(w, "message is required")
		return
	}

	req := chat.ChatRequest{
		Message:      body.Message,
		SessionID:    body.SessionID,
		SessionToken: body.SessionToken,
		UserID:       body.UserID,
		MatchCount:   body.MatchCount,
		TopicFilter:  body.TopicFilter,
		Platform:     body.Platform,
		Trusted:      middleware.IsTrusted(r.Context()),
	}

	// If request came through the Supabase-authenticated route, attach auth user.
	if u, ok := middleware.UserFromContext(r.Context()); ok {
		req.AuthUserID = &u.ID
		if req.UserID == "" {
			req.UserID = u.ID
		}
	}

	resp, err := h.svc.HandleMessage(r.Context(), req)
	if err != nil {
		if errors.Is(err, chat.ErrSessionForbidden) {
			respond.Forbidden(w)
			return
		}
		respond.InternalError(w)
		return
	}
	respond.OK(w, resp)
}

// ─── GET /api/chat/history ───────────────────────────────────────────────────
// Requires session_token query param (or Supabase JWT for authenticated users).

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		// Backward-compat alias.
		sessionID = r.URL.Query().Get("sessionId")
	}
	if sessionID == "" {
		respond.BadRequest(w, "session_id is required")
		return
	}

	token := r.URL.Query().Get("session_token")

	// Optional Supabase auth: authenticated users can access their own sessions.
	var authUserID *string
	if u, ok := middleware.UserFromContext(r.Context()); ok {
		authUserID = &u.ID
	}

	if token == "" && authUserID == nil {
		respond.Unauthorized(w)
		return
	}

	msgs, err := h.svc.GetPublicHistory(r.Context(), sessionID, token, authUserID)
	if err != nil {
		if errors.Is(err, chat.ErrSessionForbidden) {
			respond.Forbidden(w)
			return
		}
		respond.InternalError(w)
		return
	}
	respond.OK(w, map[string]any{"data": msgs})
}

// ─── POST /v1/chat/media ─────────────────────────────────────────────────────

func (h *Handler) Media(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil { // 25 MB
		respond.BadRequest(w, "failed to parse multipart form")
		return
	}

	sessionID := r.FormValue("session_id")
	userID := r.FormValue("user_id")
	msgType := r.FormValue("message_type")
	platform := r.FormValue("platform")

	if sessionID == "" {
		respond.BadRequest(w, "session_id is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respond.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")

	// message_type is optional: web clients send only the file, so infer it from
	// the MIME type (or filename extension). Falls back to "document".
	if msgType == "" {
		msgType = inferMessageType(mimeType, header.Filename)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respond.InternalError(w)
		return
	}

	req := chat.MediaRequest{
		SessionID:   sessionID,
		UserID:      userID,
		Platform:    platform,
		MessageType: msgType,
		Data:        data,
		Filename:    header.Filename,
		MimeType:    mimeType,
	}

	resp, err := h.svc.ProcessMedia(r.Context(), req)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, resp)
}

// inferMessageType maps a file's MIME type (or filename extension as a fallback)
// to one of the types ProcessMedia understands: "photo", "voice" or "document".
// Web clients upload only the file without an explicit message_type field.
func inferMessageType(mimeType, filename string) string {
	switch mt := strings.ToLower(strings.TrimSpace(mimeType)); {
	case strings.HasPrefix(mt, "image/"):
		return "photo"
	case strings.HasPrefix(mt, "audio/"):
		return "voice"
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return "photo"
	case ".ogg", ".oga", ".opus", ".mp3", ".wav", ".m4a":
		return "voice"
	default:
		return "document"
	}
}
