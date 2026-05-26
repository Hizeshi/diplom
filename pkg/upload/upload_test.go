package upload

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"photo.jpg", "photo.jpg"},
		{"../../etc/passwd", "passwd"},
		{"a/b/c.jpg", "c.jpg"},
		{"hello world.png", "hello_world.png"},
		{"名前.jpg", "__.jpg"},
		{"", "file"},
		{"..", "file"},
		{strings.Repeat("a", 200) + ".jpg", strings.Repeat("a", 116) + ".jpg"},
	}
	for _, c := range cases {
		got := SanitizeFilename(c.in)
		if got != c.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUniqueFilename(t *testing.T) {
	a := UniqueFilename("photo.jpg")
	b := UniqueFilename("photo.jpg")
	if a == b {
		t.Fatal("UniqueFilename returned same value twice")
	}
	if !strings.HasSuffix(a, "-photo.jpg") {
		t.Errorf("unexpected format: %q", a)
	}
}

func TestValidateMIME_AllowedJPEG(t *testing.T) {
	// JPEG magic bytes: FF D8 FF
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	mime, err := ValidateMIME(data, ImageMIMEs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(mime, "image/jpeg") {
		t.Errorf("expected image/jpeg, got %q", mime)
	}
}

func TestValidateMIME_Rejected(t *testing.T) {
	// ELF binary magic
	data := []byte{0x7F, 'E', 'L', 'F', 0x02}
	_, err := ValidateMIME(data, ImageMIMEs)
	if err != ErrUnsupportedMIME {
		t.Fatalf("expected ErrUnsupportedMIME, got %v", err)
	}
}

func TestValidateMIME_FakeContentType(t *testing.T) {
	// PNG header but caller claims it's audio — should still detect PNG.
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	_, err := ValidateMIME(pngHeader, AudioMIMEs)
	if err != ErrUnsupportedMIME {
		t.Fatalf("expected ErrUnsupportedMIME for PNG in audio allowlist, got %v", err)
	}
}
