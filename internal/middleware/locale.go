package middleware

import (
	"context"
	"net/http"
	"strings"
)

type localeKey struct{}

const (
	LocaleRU = "ru"
	LocaleKK = "kk"
	LocaleEN = "en"
)

var supportedLocales = map[string]bool{
	LocaleRU: true,
	LocaleKK: true,
	LocaleEN: true,
}

// Locale reads ?lang= or Accept-Language header and stores the locale in context.
// Falls back to "ru" for unsupported or missing values.
func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := strings.ToLower(r.URL.Query().Get("lang"))

		if !supportedLocales[locale] {
			// Parse first tag from Accept-Language: "kk-KZ,ru;q=0.9" → "kk"
			if lang := r.Header.Get("Accept-Language"); lang != "" {
				tag := strings.SplitN(strings.TrimSpace(strings.SplitN(lang, ",", 2)[0]), "-", 2)[0]
				locale = strings.ToLower(tag)
			}
		}

		if !supportedLocales[locale] {
			locale = LocaleRU
		}

		ctx := context.WithValue(r.Context(), localeKey{}, locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LocaleFromContext returns the locale stored by Locale middleware. Defaults to "ru".
func LocaleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(localeKey{}).(string); ok && v != "" {
		return v
	}
	return LocaleRU
}
