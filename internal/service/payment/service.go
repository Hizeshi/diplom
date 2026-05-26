package paymentsvc

import (
	"context"

	"iq-home/backend/internal/domain/payment"
)

type repo interface {
	UpdateOrderPaymentStatus(ctx context.Context, orderID int64, txID, paymentStatus, orderStatus string) error
}

type Service struct {
	repo repo
}

func New(repo repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) ProcessWebhook(ctx context.Context, p payment.WebhookPayload) error {
	var orderStatus string
	switch p.Status {
	case "success":
		orderStatus = "confirmed"
	case "failed":
		orderStatus = "cancelled"
	default:
		return payment.ErrUnknownStatus
	}
	return s.repo.UpdateOrderPaymentStatus(ctx, p.OrderID, p.TxID, p.Status, orderStatus)
}
