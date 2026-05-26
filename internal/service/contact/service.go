package contactsvc

import (
	"context"
	"fmt"

	"iq-home/backend/internal/domain/contact"
)

type repo interface {
	Create(ctx context.Context, name, email, message string) error
}

type Notifier interface {
	SendMessage(ctx context.Context, chatID, text string) error
}

type Service struct {
	repo      repo
	notifier  Notifier
	managerID string
}

func New(repo repo, notifier Notifier, managerChatID string) *Service {
	return &Service{repo: repo, notifier: notifier, managerID: managerChatID}
}

func (s *Service) Create(ctx context.Context, req contact.CreateRequest) error {
	if err := s.repo.Create(ctx, req.Name, req.Email, req.Message); err != nil {
		return fmt.Errorf("contact: create: %w", err)
	}

	if s.notifier != nil && s.managerID != "" {
		text := fmt.Sprintf(
			"📬 Новая заявка\nИмя: %s\nEmail: %s\nСообщение: %s",
			req.Name, req.Email, req.Message,
		)
		// Non-fatal: best-effort notification
		_ = s.notifier.SendMessage(ctx, s.managerID, text)
	}

	return nil
}
