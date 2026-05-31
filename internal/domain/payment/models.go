package payment

import "errors"

// WebhookPayload is the payload sent by the payment provider.
// Accepts both orderId (l-xor-pay.vercel.app) and order_id (legacy) field names.
type WebhookPayload struct {
	OrderID      int64  `json:"order_id"`
	OrderIDCamel int64  `json:"orderId"`
	Status       string `json:"status"` // "success" | "failed"
	TxID         string `json:"tx_id"`
}

// ResolvedOrderID returns whichever ID field is set.
func (p *WebhookPayload) ResolvedOrderID() int64 {
	if p.OrderID != 0 {
		return p.OrderID
	}
	return p.OrderIDCamel
}

var (
	ErrUnknownStatus  = errors.New("payment: unknown status")
	ErrOrderNotFound  = errors.New("payment: order not found")
	ErrAlreadyHandled = errors.New("payment: webhook already handled")
)
