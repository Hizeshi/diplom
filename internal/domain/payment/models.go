package payment

// WebhookPayload is the payload sent by the payment provider.
type WebhookPayload struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"` // "success" | "failed"
	TxID    string `json:"tx_id"`
}
