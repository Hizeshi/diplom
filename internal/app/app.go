package app

import (
	"context"
	"log"

	"iq-home/backend/internal/client/openai"
	"iq-home/backend/internal/client/supabase"
	"iq-home/backend/internal/client/telegram"
	"iq-home/backend/internal/client/tika"
	"iq-home/backend/internal/config"
	adminhandler "iq-home/backend/internal/handler/admin"
	chathandler "iq-home/backend/internal/handler/chat"
	contacthandler "iq-home/backend/internal/handler/contact"
	describehandler "iq-home/backend/internal/handler/describe"
	"iq-home/backend/internal/handler/health"
	paymenthandler "iq-home/backend/internal/handler/payment"
	producthandler "iq-home/backend/internal/handler/product"
	productimagehandler "iq-home/backend/internal/handler/productimage"
	productimporthandler "iq-home/backend/internal/handler/productimport"
	quotehandler "iq-home/backend/internal/handler/quote"
	telegramhandler "iq-home/backend/internal/handler/telegram"
	translatehandler "iq-home/backend/internal/handler/translate"
	userhandler "iq-home/backend/internal/handler/user"
	vectorizehandler "iq-home/backend/internal/handler/vectorize"
	"iq-home/backend/internal/infra/postgres"
	"iq-home/backend/internal/middleware"
	adminrepo "iq-home/backend/internal/repository/admin"
	chatrepo "iq-home/backend/internal/repository/chat"
	contactrepo "iq-home/backend/internal/repository/contact"
	describerepo "iq-home/backend/internal/repository/describe"
	paymentrepo "iq-home/backend/internal/repository/payment"
	productrepo "iq-home/backend/internal/repository/product"
	productimagerepo "iq-home/backend/internal/repository/productimage"
	productimportrepo "iq-home/backend/internal/repository/productimport"
	translaterepo "iq-home/backend/internal/repository/translate"
	userrepo "iq-home/backend/internal/repository/user"
	vectorizerepo "iq-home/backend/internal/repository/vectorize"
	"iq-home/backend/internal/router"
	"iq-home/backend/internal/server"
	adminsvc "iq-home/backend/internal/service/admin"
	chatsvc "iq-home/backend/internal/service/chat"
	contactsvc "iq-home/backend/internal/service/contact"
	describesvc "iq-home/backend/internal/service/describe"
	paymentsvc "iq-home/backend/internal/service/payment"
	productsvc "iq-home/backend/internal/service/product"
	productimagesvc "iq-home/backend/internal/service/productimage"
	productimportsvc "iq-home/backend/internal/service/productimport"
	quotesvc "iq-home/backend/internal/service/quote"
	telegramsvc "iq-home/backend/internal/service/telegram"
	translatesvc "iq-home/backend/internal/service/translate"
	usersvc "iq-home/backend/internal/service/user"
	vectorizesvc "iq-home/backend/internal/service/vectorize"
	"iq-home/backend/pkg/logger"
)

type App struct {
	srv *server.Server
}

func New(cfg *config.Config) *App {
	logr := logger.New()

	// ── Infra ────────────────────────────────────────────────────────────────
	db, err := postgres.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	// ── Clients ──────────────────────────────────────────────────────────────
	sb := supabase.New(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	ai := openai.New(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey)
	tk := tika.New(cfg.TikaURL)

	// Single embedder backed by OpenAI — used by all services that need vectors.
	embedder := openai.NewEmbedAdapter(ai, cfg.OpenAIEmbeddingModel)

	var tg *telegram.Client
	if cfg.TelegramBotToken != "" {
		tg = telegram.New(cfg.TelegramBaseURL, cfg.TelegramBotToken)
	}

	// ── Products ─────────────────────────────────────────────────────────────
	pRepo := productrepo.New(db)
	productService := productsvc.New(pRepo, embedder)

	// ── Users ────────────────────────────────────────────────────────────────
	userService := usersvc.New(userrepo.New(db), sb, cfg.PaymentBaseURL)

	// ── Chat ─────────────────────────────────────────────────────────────────
	chatService := chatsvc.New(
		chatrepo.New(db),
		pRepo,
		embedder,
		openai.NewChatAdapter(ai),
		tk,
		sb,
		logr,
		chatsvc.Config{
			OpenAIModel:           cfg.OpenAIModel,
			OpenAIVisionModel:     cfg.OpenAIVisionModel,
			OpenAITranscribeModel: cfg.OpenAITranscribeModel,
			ProductURL:            cfg.ProductURL,
		},
	)

	// ── Admin ────────────────────────────────────────────────────────────────
	adminService := adminsvc.New(adminrepo.New(db), embedder, sb)

	// ── Contact ──────────────────────────────────────────────────────────────
	var contactNotifier contactsvc.Notifier
	if tg != nil {
		contactNotifier = tg
	}
	contactService := contactsvc.New(contactrepo.New(db), contactNotifier, cfg.ManagerChatID)

	// ── Payment ──────────────────────────────────────────────────────────────
	paymentService := paymentsvc.New(paymentrepo.New(db))

	// ── Product Images ────────────────────────────────────────────────────────
	productImageService := productimagesvc.New(productimagerepo.New(db), sb)

	// ── Product Import ───────────────────────────────────────────────────────
	productImportService := productimportsvc.New(productimportrepo.New(db))

	// ── Quote ────────────────────────────────────────────────────────────────
	quoteService := quotesvc.New(sb)

	// ── Vectorize ────────────────────────────────────────────────────────────
	vectorizeService := vectorizesvc.New(vectorizerepo.New(db), embedder, logr)

	// ── Describe ─────────────────────────────────────────────────────────────
	describeService := describesvc.New(describerepo.New(db), ai, cfg.OpenAIModel, logr)

	// ── Translate (i18n bulk) ─────────────────────────────────────────────────
	translateService := translatesvc.New(translaterepo.New(db), ai, cfg.OpenAIModel)

	// ── Telegram ─────────────────────────────────────────────────────────────
	var telegramService *telegramsvc.Service
	var telegramHandler *telegramhandler.Handler
	if tg != nil {
		telegramService = telegramsvc.New(chatService, tg, cfg.TelegramWebhookSecret)
		telegramHandler = telegramhandler.New(telegramService, cfg.TelegramWebhookSecret)
	} else {
		telegramHandler = telegramhandler.New(nil, "")
	}

	// ── Supabase auth middleware ──────────────────────────────────────────────
	sbAuth := middleware.SupabaseAuth(func(ctx context.Context, token string) (string, string, error) {
		u, err := sb.GetUser(ctx, token)
		if err != nil {
			return "", "", err
		}
		return u.ID, u.Email, nil
	})

	// ── Handlers ─────────────────────────────────────────────────────────────
	handlers := router.Handlers{
		Health:        health.New(),
		Product:       producthandler.New(productService),
		User:          userhandler.New(userService),
		Chat:          chathandler.New(chatService),
		Admin:         adminhandler.New(adminService),
		Contact:       contacthandler.New(contactService),
		Payment:       paymenthandler.New(paymentService, cfg.PaymentWebhookSecret),
		ProductImage:  productimagehandler.New(productImageService),
		ProductImport: productimporthandler.New(productImportService),
		Quote:         quotehandler.New(quoteService),
		Vectorize:     vectorizehandler.New(vectorizeService, logr),
		Describe:      describehandler.New(describeService, logr),
		Telegram:      telegramHandler,
		Translate:     translatehandler.New(translateService),
		SupabaseAuth:  sbAuth,
	}

	h := router.New(cfg, logr, handlers)
	srv := server.New(cfg.HTTPAddr, h)

	return &App{srv: srv}
}

func (a *App) Run() error {
	return a.srv.Run()
}
