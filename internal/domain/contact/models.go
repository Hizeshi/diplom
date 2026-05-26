package contact

import "time"

type Contact struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRequest struct {
	Name    string `json:"name"    validate:"required,min=2,max=100"`
	Email   string `json:"email"   validate:"required,email,max=200"`
	Message string `json:"message" validate:"required,min=5,max=2000"`
}
