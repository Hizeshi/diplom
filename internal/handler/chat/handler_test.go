package chathandler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"iq-home/backend/internal/domain/chat"
)

func TestInferMessageType(t *testing.T) {
	cases := []struct {
		name     string
		mime     string
		filename string
		want     string
	}{
		{"image mime", "image/jpeg", "photo.jpg", "photo"},
		{"image mime uppercase", "IMAGE/PNG", "p.png", "photo"},
		{"audio mime", "audio/ogg", "voice.ogg", "voice"},
		{"pdf mime", "application/pdf", "doc.pdf", "document"},
		{"image by ext only", "", "scan.WEBP", "photo"},
		{"voice by ext only", "application/octet-stream", "note.opus", "voice"},
		{"unknown ext", "", "report.xlsx", "document"},
		{"empty everything", "", "", "document"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferMessageType(tc.mime, tc.filename); got != tc.want {
				t.Fatalf("inferMessageType(%q,%q) = %q, want %q", tc.mime, tc.filename, got, tc.want)
			}
		})
	}
}

// stubService captures the request passed to ProcessMedia.
type stubService struct {
	gotMedia chat.MediaRequest
}

func (s *stubService) HandleMessage(ctx context.Context, req chat.ChatRequest) (*chat.ChatResponse, error) {
	return &chat.ChatResponse{}, nil
}

func (s *stubService) GetPublicHistory(ctx context.Context, sessionID, token string, authUserID *string) ([]chat.PublicMessage, error) {
	return nil, nil
}

func (s *stubService) ProcessMedia(ctx context.Context, req chat.MediaRequest) (*chat.ChatResponse, error) {
	s.gotMedia = req
	return &chat.ChatResponse{Answer: "ok"}, nil
}

// newMediaRequest builds a multipart request that mimics the frontend: it sends
// only session_id and a file, with NO message_type field.
func newMediaRequest(t *testing.T, sessionID, filename, contentType string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if sessionID != "" {
		_ = w.WriteField("session_id", sessionID)
	}
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	_, _ = part.Write([]byte{0x89, 0x50, 0x4e, 0x47}) // dummy bytes
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/media", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestMediaInfersMessageTypeWhenMissing(t *testing.T) {
	svc := &stubService{}
	h := New(svc)

	req := newMediaRequest(t, "user:abc", "photo.jpg", "image/jpeg")
	rec := httptest.NewRecorder()
	h.Media(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if svc.gotMedia.MessageType != "photo" {
		t.Fatalf("MessageType = %q, want %q", svc.gotMedia.MessageType, "photo")
	}
	if svc.gotMedia.SessionID != "user:abc" {
		t.Fatalf("SessionID = %q, want %q", svc.gotMedia.SessionID, "user:abc")
	}
}

func TestMediaRequiresSessionID(t *testing.T) {
	svc := &stubService{}
	h := New(svc)

	req := newMediaRequest(t, "", "photo.jpg", "image/jpeg")
	rec := httptest.NewRecorder()
	h.Media(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Fatalf("expected error message, got %s", rec.Body.String())
	}
}
