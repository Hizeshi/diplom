package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// Server
	HTTPAddr string

	// Database
	DatabaseURL string

	// Auth
	InternalToken string
	AdminUsername string
	AdminPassword string

	// Supabase
	SupabaseURL            string
	SupabaseServiceRoleKey string

	// OpenAI
	OpenAIBaseURL          string
	OpenAIAPIKey           string
	OpenAIModel            string
	OpenAIVisionModel      string
	OpenAITranscribeModel  string
	OpenAIEmbeddingModel   string

	// Tika (document parsing)
	TikaURL string

	// Telegram
	TelegramBotToken      string
	TelegramWebhookSecret string
	TelegramBaseURL       string
	ManagerChatIDs        []string

	// CORS
	CORSAllowOrigin string

	// Payment
	PaymentBaseURL        string
	PaymentWebhookSecret  string

	// Product page base URL (used in chat for product links)
	ProductURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		OpenAIBaseURL:        getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
		OpenAIModel:           getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIVisionModel:     getEnv("OPENAI_VISION_MODEL", "gpt-4o-mini"),
		OpenAITranscribeModel: getEnv("OPENAI_TRANSCRIBE_MODEL", "gpt-4o-mini-transcribe"),
		OpenAIEmbeddingModel:  getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-large"),
		TikaURL:              getEnv("TIKA_URL", "http://127.0.0.1:9998"),
		TelegramBaseURL:      getEnv("TELEGRAM_BASE_URL", "https://api.telegram.org"),
		CORSAllowOrigin:      getEnv("CORS_ALLOW_ORIGIN", "*"),
		PaymentBaseURL:       os.Getenv("PAYMENT_BASE_URL"),
		PaymentWebhookSecret: os.Getenv("PAYMENT_WEBHOOK_SECRET"),
		ProductURL:           getEnv("PRODUCT_URL", "https://iq-home.kz/products/"),

		DatabaseURL:            os.Getenv("DATABASE_URL"),
		InternalToken:          os.Getenv("INTERNAL_TOKEN"),
		AdminUsername:          os.Getenv("ADMIN_USERNAME"),
		AdminPassword:          os.Getenv("ADMIN_PASSWORD"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		OpenAIAPIKey:           os.Getenv("OPENAI_API_KEY"),
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		ManagerChatIDs:        parseIDs(os.Getenv("MANAGER_CHAT_IDS")),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":              c.DatabaseURL,
		"INTERNAL_TOKEN":            c.InternalToken,
		"SUPABASE_URL":              c.SupabaseURL,
		"SUPABASE_SERVICE_ROLE_KEY": c.SupabaseServiceRoleKey,
		"OPENAI_API_KEY":            c.OpenAIAPIKey,
		"PAYMENT_WEBHOOK_SECRET":    c.PaymentWebhookSecret,
		"ADMIN_USERNAME":            c.AdminUsername,
		"ADMIN_PASSWORD":            c.AdminPassword,
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("required env var %s is not set", name)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseIDs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if id := strings.TrimSpace(p); id != "" {
			out = append(out, id)
		}
	}
	return out
}
