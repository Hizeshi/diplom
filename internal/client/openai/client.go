package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ─── Types ───────────────────────────────────────────────────────────────────

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string | []ContentPart
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type CompleteOptions struct {
	Model       string
	Messages    []Message
	JSONMode    bool
	Temperature *float64
	MaxTokens   int
}

// ─── Chat Completions ────────────────────────────────────────────────────────

// Complete sends messages to the chat completions API and returns the response text.
func (c *Client) Complete(ctx context.Context, opts CompleteOptions) (string, error) {
	body := map[string]any{
		"model":    opts.Model,
		"messages": opts.Messages,
	}
	if opts.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if opts.MaxTokens > 0 {
		body["max_completion_tokens"] = opts.MaxTokens
	}
	if opts.Temperature != nil {
		body["temperature"] = *opts.Temperature
	}

	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := c.post(ctx, "/v1/chat/completions", body, &resp); err != nil {
		return "", fmt.Errorf("openai: complete: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("openai: complete: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: complete: empty response")
	}
	return resp.Choices[0].Message.Content, nil
}

// ─── Transcription ───────────────────────────────────────────────────────────

// Transcribe converts audio bytes to text using the Whisper API.
func (c *Client) Transcribe(ctx context.Context, model string, audio []byte, filename string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	if err := mw.WriteField("model", model); err != nil {
		return "", err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai: transcribe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai: transcribe: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("openai: transcribe: decode: %w", err)
	}
	return result.Text, nil
}

// ─── Embeddings ──────────────────────────────────────────────────────────────

// Embed generates a 1024-dimensional embedding for the given text.
// It uses the model configured via SetEmbeddingModel (default: text-embedding-3-large).
func (c *Client) Embed(ctx context.Context, model, text string) ([]float32, error) {
	req := map[string]any{
		"model":      model,
		"input":      text,
		"dimensions": 1024,
	}
	var resp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := c.post(ctx, "/v1/embeddings", req, &resp); err != nil {
		return nil, fmt.Errorf("openai: embed: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("openai: embed: %s", resp.Error.Message)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai: embed: empty response")
	}
	return resp.Data[0].Embedding, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, b)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
