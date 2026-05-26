// Package upload provides helpers for safe file uploads:
// filename sanitization and MIME-type validation.
package upload

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrUnsupportedMIME is returned when a file's detected content type is not
// in the caller's allowlist.
var ErrUnsupportedMIME = errors.New("upload: unsupported file type")

// Allowlists for common upload categories.
var (
	ImageMIMEs = []string{
		"image/jpeg", "image/png", "image/webp", "image/gif",
	}
	AudioMIMEs = []string{
		"audio/ogg", "audio/mpeg", "audio/mp4", "audio/webm",
		"audio/x-wav", "audio/wav",
	}
	DocumentMIMEs = []string{
		"application/pdf", "text/plain",
	}
)

var safeNameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// SanitizeFilename returns a safe, path-traversal-free version of name.
// All characters outside [a-zA-Z0-9._-] are replaced with "_".
// The result is capped at 120 chars. An empty name becomes "file".
func SanitizeFilename(name string) string {
	// Strip directory components.
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "/")
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		name = name[idx+1:]
	}

	name = safeNameRe.ReplaceAllString(name, "_")
	// Reject names that are empty, dot-only, or pure underscores after sanitizing.
	stripped := strings.TrimRight(strings.ReplaceAll(name, "_", ""), ".")
	if stripped == "" {
		name = "file"
	}
	if len(name) > 120 {
		ext := filepath.Ext(name)
		base := name[:120-len(ext)]
		name = base + ext
	}
	return name
}

// UniqueFilename returns "{randomHex}-{sanitized(name)}", guaranteeing a
// collision-free storage path regardless of what the client sent.
func UniqueFilename(name string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) + "-" + SanitizeFilename(name)
}

// DetectMIME sniffs the first 512 bytes of data and returns the detected
// MIME type. Falls back to "application/octet-stream" for binary data.
func DetectMIME(data []byte) string {
	n := 512
	if len(data) < n {
		n = len(data)
	}
	return http.DetectContentType(data[:n])
}

// ValidateMIME detects the actual MIME type of data and checks it against
// allowed. Returns ErrUnsupportedMIME if the detected type is not in the list.
// Also returns the detected MIME type for the caller to use.
func ValidateMIME(data []byte, allowed []string) (string, error) {
	detected := DetectMIME(data)
	for _, a := range allowed {
		if strings.HasPrefix(detected, a) {
			return detected, nil
		}
	}
	return detected, ErrUnsupportedMIME
}
