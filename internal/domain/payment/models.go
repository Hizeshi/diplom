package payment

import "errors"

// WebhookPayload is the payload sent by the payment provider.
type WebhookPayload struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"` // "success" | "failed"
	TxID    string `json:"tx_id"`
}

var (
	ErrUnknownStatus  = errors.New("payment: unknown status")
	ErrOrderNotFound  = errors.New("payment: order not found")
	ErrAlreadyHandled = errors.New("payment: webhook already handled")
)
