package chathandler

import (
	"context"
	"io"
	"net/http"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/internal/middleware"
	"iq-home/backend/pkg/respond"
	"iq-home/backend/pkg/validate"
)

type service interface {
	HandleMessage(ctx context.Context, req chat.ChatRequest) (*chat.ChatResponse, error)
	GetPublicHistory(ctx context.Context, sessionID string) ([]chat.PublicMessage, error)
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
		Message     string `json:"message"      validate:"required,min=1,max=2000"`
		SessionID   string `json:"session_id"   validate:"required,max=100"`
		UserID      string `json:"user_id"`
		MatchCount  int    `json:"match_count"`
		TopicFilter string `json:"topic_filter"`
		Platform    string `json:"platform"`
	}
	if !validate.DecodeAndValidate(w, r, &body) {
		return
	}

	req := chat.ChatRequest{
		Message:     body.Message,
		SessionID:   body.SessionID,
		UserID:      body.UserID,
		MatchCount:  body.MatchCount,
		TopicFilter: body.TopicFilter,
		Platform:    body.Platform,
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
		respond.InternalError(w)
		return
	}
	respond.OK(w, resp)
}

// ─── GET /api/chat/history ───────────────────────────────────────────────────

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		respond.BadRequest(w, "sessionId is required")
		return
	}

	msgs, err := h.svc.GetPublicHistory(r.Context(), sessionID)
	if err != nil {
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

	if sessionID == "" || msgType == "" {
		respond.BadRequest(w, "session_id and message_type are required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respond.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

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
		MimeType:    header.Header.Get("Content-Type"),
	}

	resp, err := h.svc.ProcessMedia(r.Context(), req)
	if err != nil {
		respond.InternalError(w)
		return
	}
	respond.OK(w, resp)
}
