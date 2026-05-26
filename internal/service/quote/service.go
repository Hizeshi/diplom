package quotesvc

import (
	"context"
	"fmt"
	"time"

	"iq-home/backend/internal/domain/quote"
	"iq-home/backend/internal/infra/pdf"
)

const bucket = "quotes"

type storage interface {
	Upload(ctx context.Context, bucket, objectPath string, data []byte, contentType string) (string, error)
}

type Service struct {
	storage storage
}

func New(storage storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) Generate(ctx context.Context, req quote.Request) (*quote.Response, error) {
	data, err := pdf.Generate(req)
	if err != nil {
		return nil, fmt.Errorf("quote: generate pdf: %w", err)
	}

	objectPath := fmt.Sprintf("%s/quote.pdf", time.Now().UTC().Format("2006/01/02-150405"))
	url, err := s.storage.Upload(ctx, bucket, objectPath, data, "application/pdf")
	if err != nil {
		return nil, fmt.Errorf("quote: upload: %w", err)
	}

	return &quote.Response{URL: url}, nil
}
