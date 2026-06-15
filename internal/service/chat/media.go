package chatsvc

import (
	"context"
	"fmt"
	"strings"

	"iq-home/backend/internal/domain/chat"
	"iq-home/backend/pkg/upload"
)

// ProcessMedia handles voice/photo/document uploads and returns a ChatResponse.
func (s *Service) ProcessMedia(ctx context.Context, req chat.MediaRequest) (*chat.ChatResponse, error) {
	var (
		searchMessage string
		filePath      string
		err           error
	)

	switch req.MessageType {
	case "voice":
		searchMessage, err = s.processVoice(ctx, req.Data, req.Filename)
	case "photo":
		searchMessage, err = s.processPhoto(ctx, req.Data, req.MimeType)
	case "document":
		searchMessage, err = s.processDocument(ctx, req.Data, req.MimeType)
	default:
		return nil, fmt.Errorf("media: unknown message type: %s", req.MessageType)
	}
	if err != nil {
		return nil, fmt.Errorf("media: process %s: %w", req.MessageType, err)
	}

	// Upload attachment to storage.
	// file_path stores only the relative object key (not the full URL), because
	// the frontend constructs the public URL itself. The full URL goes into
	// meta.attachment_url so the frontend can use it directly without re-building it.
	meta := map[string]any{
		"message_type":  req.MessageType,
		"original_text": searchMessage,
	}
	if len(req.Data) > 0 {
		objectPath := fmt.Sprintf("%s/%s", req.SessionID, upload.UniqueFilename(req.Filename))
		if publicURL, uploadErr := s.store.Upload(ctx, "chat-attachments", objectPath, req.Data, req.MimeType); uploadErr == nil {
			filePath = objectPath
			meta["attachment_url"] = publicURL
			meta["attachment_name"] = req.Filename
		}
	}

	// Forward to main chat flow. HandleMessage is responsible for saving both
	// the user message (with media metadata) and the assistant response.
	// Do NOT call SaveMessage here — that would create duplicates.
	chatReq := chat.ChatRequest{
		Message:     searchMessage,
		SessionID:   req.SessionID,
		AuthUserID:  req.AuthUserID,
		UserID:      req.UserID,
		Platform:    req.Platform,
		MatchCount:  5,
		Trusted:     true, // media endpoint is always /v1 (internal)
		MessageType: req.MessageType,
		FilePath:    filePath,
		MetaData:    meta,
	}

	return s.HandleMessage(ctx, chatReq)
}

// processVoice transcribes audio using Whisper.
func (s *Service) processVoice(ctx context.Context, data []byte, filename string) (string, error) {
	if filename == "" {
		filename = "audio.ogg"
	}
	// Whisper rejects .oga but accepts .ogg — they are the same OGG container.
	if strings.HasSuffix(strings.ToLower(filename), ".oga") {
		filename = filename[:len(filename)-4] + ".ogg"
	}
	text, err := s.llm.Transcribe(ctx, s.cfg.OpenAITranscribeModel, data, filename)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	return strings.TrimSpace(text), nil
}

// processPhoto sends the image to the vision model and returns a search query.
func (s *Service) processPhoto(ctx context.Context, data []byte, mimeType string) (string, error) {
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	description, err := s.analyzeImage(ctx, data, mimeType)
	if err != nil {
		return "", fmt.Errorf("vision: %w", err)
	}
	// Prefix so the chat service knows this came from image analysis.
	return "bot: " + description, nil
}

// processDocument extracts text using Apache Tika.
// If Tika is unavailable, falls back to a generic acknowledgement so the user
// still gets a chat response instead of a 500.
func (s *Service) processDocument(ctx context.Context, data []byte, mimeType string) (string, error) {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	text, err := s.parser.ExtractText(ctx, data, mimeType)
	if err != nil {
		// Tika not available — return a neutral prompt so the LLM can reply.
		return "Пользователь прислал документ. Текст не удалось извлечь. Попроси прислать содержимое текстом.", nil
	}
	text = strings.TrimSpace(text)
	if len(text) > 2000 {
		text = text[:2000]
	}
	return "bot: " + text, nil
}

