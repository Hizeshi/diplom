package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	serviceKey string
	http       *http.Client
}

func New(baseURL, serviceKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

// ─── Auth ────────────────────────────────────────────────────────────────────

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// GetUser validates a Bearer token and returns the authenticated user.
func (c *Client) GetUser(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", c.serviceKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("supabase: get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase: get user: status %d", resp.StatusCode)
	}

	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("supabase: get user: decode: %w", err)
	}
	return &u, nil
}

// AdminDeleteUser removes a user from Supabase Auth.
func (c *Client) AdminDeleteUser(ctx context.Context, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/auth/v1/admin/users/"+userID, nil)
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("supabase: delete user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("supabase: delete user: status %d", resp.StatusCode)
	}
	return nil
}

// AdminUpdateEmail changes a user's email in Supabase Auth.
func (c *Client) AdminUpdateEmail(ctx context.Context, userID, email string) error {
	body, _ := json.Marshal(map[string]string{"email": email})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/auth/v1/admin/users/"+userID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("supabase: update email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("supabase: update email: status %d", resp.StatusCode)
	}
	return nil
}

// ─── Storage ─────────────────────────────────────────────────────────────────

// Upload puts a file into Supabase Storage and returns the public URL.
func (c *Client) Upload(ctx context.Context, bucket, objectPath string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, bucket, objectPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	c.setServiceHeaders(req)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase: upload %s/%s: %w", bucket, objectPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase: upload %s/%s: status %d: %s", bucket, objectPath, resp.StatusCode, body)
	}

	return c.PublicURL(bucket, objectPath), nil
}

// Delete removes one or more objects from Supabase Storage.
func (c *Client) Delete(ctx context.Context, bucket string, paths []string) error {
	body, _ := json.Marshal(map[string][]string{"prefixes": paths})
	url := fmt.Sprintf("%s/storage/v1/object/%s", c.baseURL, bucket)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("supabase: delete from %s: %w", bucket, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("supabase: delete from %s: status %d", bucket, resp.StatusCode)
	}
	return nil
}

// PublicURL returns the public URL for a stored object.
func (c *Client) PublicURL(bucket, objectPath string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.baseURL, bucket, objectPath)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (c *Client) setServiceHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
}
