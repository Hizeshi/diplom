package openai

import (
	"context"

	chatsvc         "iq-home/backend/internal/service/chat"
	compatibilitysvc "iq-home/backend/internal/service/compatibility"
)

// ChatAdapter wraps Client to satisfy chatsvc.llm interface.
type ChatAdapter struct {
	c *Client
}

func NewChatAdapter(c *Client) *ChatAdapter {
	return &ChatAdapter{c: c}
}

func (a *ChatAdapter) Complete(ctx context.Context, opts chatsvc.CompleteOptions) (string, error) {
	msgs := make([]Message, len(opts.Messages))
	for i, m := range opts.Messages {
		switch v := m.Content.(type) {
		case string:
			msgs[i] = Message{Role: m.Role, Content: v}
		case []chatsvc.ContentPart:
			parts := make([]ContentPart, len(v))
			for j, p := range v {
				parts[j] = ContentPart{
					Type: p.Type,
					Text: p.Text,
				}
				if p.ImageURL != "" {
					parts[j].ImageURL = &ImageURL{URL: p.ImageURL, Detail: "auto"}
				}
			}
			msgs[i] = Message{Role: m.Role, Content: parts}
		}
	}

	return a.c.Complete(ctx, CompleteOptions{
		Model:    opts.Model,
		Messages: msgs,
		JSONMode: opts.JSONMode,
	})
}

func (a *ChatAdapter) Transcribe(ctx context.Context, model string, data []byte, filename string) (string, error) {
	return a.c.Transcribe(ctx, model, data, filename)
}

// CompatibilityAdapter wraps Client to satisfy compatibilitysvc.llm interface.
type CompatibilityAdapter struct {
	c *Client
}

func NewCompatibilityAdapter(c *Client) *CompatibilityAdapter {
	return &CompatibilityAdapter{c: c}
}

func (a *CompatibilityAdapter) Complete(ctx context.Context, opts compatibilitysvc.CompleteOptions) (string, error) {
	msgs := make([]Message, len(opts.Messages))
	for i, m := range opts.Messages {
		if s, ok := m.Content.(string); ok {
			msgs[i] = Message{Role: m.Role, Content: s}
		}
	}
	return a.c.Complete(ctx, CompleteOptions{
		Model:    opts.Model,
		Messages: msgs,
		JSONMode: opts.JSONMode,
	})
}

// EmbedAdapter wraps Client to satisfy a simple Embed(ctx, text) interface.
type EmbedAdapter struct {
	c     *Client
	model string
}

func NewEmbedAdapter(c *Client, model string) *EmbedAdapter {
	return &EmbedAdapter{c: c, model: model}
}

func (a *EmbedAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.c.Embed(ctx, a.model, text)
}
