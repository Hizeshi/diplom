package telegramsvc

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"iq-home/backend/internal/domain/chat"
)

// Update is a Telegram Bot API Update object (minimal fields we care about).
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Voice     *File  `json:"voice"`
	Photo     []File `json:"photo"` // array, last is highest resolution
	Document  *File  `json:"document"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type File struct {
	FileID   string `json:"file_id"`
	MimeType string `json:"mime_type"`
	FileName string `json:"file_name"`
}

// chatService is satisfied by chatsvc.Service.
type chatService interface {
	HandleMessage(ctx context.Context, req chat.ChatRequest) (*chat.ChatResponse, error)
	ProcessMedia(ctx context.Context, req chat.MediaRequest) (*chat.ChatResponse, error)
}

// telegramClient is satisfied by telegram.Client.
type telegramClient interface {
	SendMessage(ctx context.Context, chatID, text string) error
	GetFile(ctx context.Context, fileID string) ([]byte, string, error)
}

type Service struct {
	chat   chatService
	tg     telegramClient
	secret string
}

func New(chat chatService, tg telegramClient, webhookSecret string) *Service {
	return &Service{chat: chat, tg: tg, secret: webhookSecret}
}

// HandleUpdate dispatches a Telegram update to the chat service and replies.
func (s *Service) HandleUpdate(ctx context.Context, upd Update) error {
	msg := upd.Message
	if msg == nil {
		return nil
	}

	chatIDStr := fmt.Sprintf("%d", msg.Chat.ID)
	sessionID := "tg-" + chatIDStr

	var resp *chat.ChatResponse
	var err error

	switch {
	case msg.Voice != nil:
		data, filePath, ferr := s.tg.GetFile(ctx, msg.Voice.FileID)
		if ferr != nil {
			return ferr
		}
		ct := msg.Voice.MimeType
		if ct == "" {
			ct = "audio/ogg"
		}
		resp, err = s.chat.ProcessMedia(ctx, chat.MediaRequest{
			SessionID:   sessionID,
			MessageType: "voice",
			Filename:    filepath.Base(filePath),
			Data:        data,
			MimeType:    ct,
			Platform:    "telegram",
		})

	case len(msg.Photo) > 0:
		photo := msg.Photo[len(msg.Photo)-1]
		data, filePath, ferr := s.tg.GetFile(ctx, photo.FileID)
		if ferr != nil {
			return ferr
		}
		resp, err = s.chat.ProcessMedia(ctx, chat.MediaRequest{
			SessionID:   sessionID,
			MessageType: "photo",
			Filename:    filepath.Base(filePath),
			Data:        data,
			MimeType:    "image/jpeg",
			Platform:    "telegram",
		})

	case msg.Document != nil:
		data, filePath, ferr := s.tg.GetFile(ctx, msg.Document.FileID)
		if ferr != nil {
			return ferr
		}
		ct := msg.Document.MimeType
		if ct == "" {
			ct = "application/octet-stream"
		}
		resp, err = s.chat.ProcessMedia(ctx, chat.MediaRequest{
			SessionID:   sessionID,
			MessageType: "document",
			Filename:    filepath.Base(filePath),
			Data:        data,
			MimeType:    ct,
			Platform:    "telegram",
		})

	default:
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			return nil
		}
		resp, err = s.chat.HandleMessage(ctx, chat.ChatRequest{
			SessionID: sessionID,
			Message:   text,
			UserID:    chatIDStr,
			Platform:  "telegram",
		})
	}

	if err != nil {
		return fmt.Errorf("telegram: handle message: %w", err)
	}

	if resp == nil || resp.Answer == "" {
		return nil
	}

	return s.tg.SendMessage(ctx, chatIDStr, resp.Answer)
}
