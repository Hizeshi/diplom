package tika

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ExtractText sends a document to Apache Tika and returns the extracted plain text.
func (c *Client) ExtractText(ctx context.Context, data []byte, contentType string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/tika", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("tika: extract: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tika: extract: status %d: %s", resp.StatusCode, b)
	}

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("tika: extract: read: %w", err)
	}

	return string(text), nil
}
