package telegram

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
	baseURL string // e.g. https://api.telegram.org
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// SendMessage sends a plain text message to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	body, _ := json.Marshal(map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	return c.apiPost(ctx, "sendMessage", "application/json", bytes.NewReader(body))
}

// SendDocument sends a file to a chat.
func (c *Client) SendDocument(ctx context.Context, chatID, filename string, data []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("chat_id", chatID); err != nil {
		return err
	}
	fw, err := mw.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	mw.Close()

	return c.apiPost(ctx, "sendDocument", mw.FormDataContentType(), &buf)
}

// SendChatAction sends a typing indicator or similar action.
func (c *Client) SendChatAction(ctx context.Context, chatID, action string) error {
	body, _ := json.Marshal(map[string]string{
		"chat_id": chatID,
		"action":  action,
	})
	return c.apiPost(ctx, "sendChatAction", "application/json", bytes.NewReader(body))
}

// GetFile downloads a file by its Telegram file_id.
// Returns the raw bytes and the file path (extension included).
func (c *Client) GetFile(ctx context.Context, fileID string) ([]byte, string, error) {
	// Step 1: resolve file_path
	body, _ := json.Marshal(map[string]string{"file_id": fileID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.apiURL("getFile"), bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: getFile: %w", err)
	}
	defer resp.Body.Close()

	var meta struct {
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, "", fmt.Errorf("telegram: getFile: decode: %w", err)
	}

	// Step 2: download
	filePath := meta.Result.FilePath
	downloadURL := fmt.Sprintf("%s/file/bot%s/%s", c.baseURL, c.token, filePath)

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp2, err := c.http.Do(req2)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: download file: %w", err)
	}
	defer resp2.Body.Close()

	data, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: read file: %w", err)
	}

	return data, filePath, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (c *Client) apiURL(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
}

func (c *Client) apiPost(ctx context.Context, method, contentType string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL(method), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: %s: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: %s: status %d: %s", method, resp.StatusCode, b)
	}
	return nil
}
