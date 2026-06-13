package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"

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
	"iq-home/backend/internal/middleware"
)

type Handlers struct {
	Health        *health.Handler
	Product       *producthandler.Handler
	User          *userhandler.Handler
	Chat          *chathandler.Handler
	Admin         *adminhandler.Handler
	Contact       *contacthandler.Handler
	Payment       *paymenthandler.Handler
	ProductImage  *productimagehandler.Handler
	ProductImport *productimporthandler.Handler
	Quote         *quotehandler.Handler
	Telegram      *telegramhandler.Handler
	Vectorize     *vectorizehandler.Handler
	Describe      *describehandler.Handler
	Translate     *translatehandler.Handler

	// SupabaseAuth is the middleware applied to /api/user/* routes.
	SupabaseAuth func(http.Handler) http.Handler
}

func New(cfg *config.Config, log *slog.Logger, h Handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery(log))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(log))
	r.Use(middleware.CORS(cfg.CORSAllowOrigin))

	// Public
	r.Get("/health", h.Health.Health)

	// Public API
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.Locale) // detect lang from ?lang= or Accept-Language

		// Products (public)
		r.Get("/filters", h.Product.Filters)
		r.Get("/products/{id}", h.Product.GetByID)
		r.Post("/products/search", h.Product.Search)

		// Contact — rate limited
		r.With(middleware.RateLimit(rate.Limit(5), 10)).Post("/contact", h.Contact.Create)

		// Chat — requires auth
		r.Group(func(r chi.Router) {
			r.Use(h.SupabaseAuth)
			r.Get("/chat/history", h.Chat.History)
			r.With(middleware.RateLimit(rate.Limit(5), 10)).Post("/chat", h.Chat.Chat)
		})

		// Payment webhooks
		r.Post("/payment/webhook", h.Payment.Process)                                              // HMAC-signed (legacy provider)
		r.With(middleware.RateLimit(rate.Limit(10), 20)).Post("/payment/notify", h.Payment.Notify) // l-xor-pay.vercel.app

		// User (Supabase JWT required)
		r.Route("/user", func(r chi.Router) {
			r.Use(h.SupabaseAuth)

			r.Get("/cart", h.User.GetCart)
			r.Post("/cart", h.User.AddToCart)
			r.Put("/cart/{productId}", h.User.UpdateCartQuantity)
			r.Delete("/cart/{productId}", h.User.RemoveFromCart)

			r.Get("/favorites", h.User.GetFavorites)
			r.Post("/favorites", h.User.ToggleFavorite)
			r.Delete("/favorites/{productId}", h.User.RemoveFavorite)

			r.Get("/history", h.User.GetHistory)
			r.Post("/history", h.User.AddHistory)
			r.Get("/recommendations", h.User.GetRecommendations)

			r.Get("/orders", h.User.GetOrders)
			r.Get("/orders/{id}", h.User.GetOrderDetail)

			r.Post("/checkout", h.User.Checkout)
			r.Get("/session", h.User.GetSession)

			r.Post("/avatar", h.User.UpdateAvatar)
			r.Delete("/avatar", h.User.DeleteAvatar)
		})

		// Admin (BasicAuth)
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.BasicAuth(cfg.AdminUsername, cfg.AdminPassword))
			r.Use(middleware.AdminAudit(log))

			r.Get("/chats", h.Admin.ListChats)
			r.Get("/chats/{sessionId}", h.Admin.GetChatHistory)
			r.Post("/chats/{sessionId}/toggle", h.Admin.ToggleHumanMode)
			r.Post("/chats/{sessionId}/message", h.Admin.SendManagerMessage)

			r.Get("/products", h.Admin.ListProducts)
			r.Post("/products", h.Admin.CreateProduct)
			r.Post("/products/scan-duplicates", h.Admin.ScanDuplicates)
			r.Get("/products/{id}", h.Admin.GetProduct)
			r.Put("/products/{id}", h.Admin.UpdateProduct)
			r.Put("/products/{id}/configurator", h.Admin.UpdateConfiguratorType)
			r.Delete("/products/{id}", h.Admin.DeleteProduct)
			r.Put("/products/{id}/i18n", h.Product.UpdateProductI18n)

			// i18n for reference tables
			r.Put("/series/{id}/i18n", h.Product.UpdateSeriesI18n)
			r.Put("/brands/{id}/i18n", h.Product.UpdateBrandI18n)
			r.Put("/colors/{id}/i18n", h.Product.UpdateColorI18n)

			r.Get("/users", h.Admin.ListUsers)
			r.Get("/users/{id}", h.Admin.GetUserDetail)
			r.Put("/users/{id}", h.Admin.UpdateUser)
			r.Delete("/users/{id}", h.Admin.DeleteUser)
			r.Delete("/users/{id}/cart", h.Admin.ClearUserCart)
			r.Delete("/users/{id}/cart/{productId}", h.Admin.DeleteCartItem)
			r.Delete("/users/{id}/history", h.Admin.ClearUserHistory)
			r.Delete("/users/{id}/history/{productId}", h.Admin.DeleteHistoryItem)
			r.Delete("/users/{id}/favorites", h.Admin.ClearUserFavorites)
			r.Delete("/users/{id}/favorites/{productId}", h.Admin.DeleteFavoriteItem)

			r.Get("/orders", h.Admin.ListOrders)
			r.Get("/orders/{id}", h.Admin.GetOrderDetail)
			r.Put("/orders/{id}/status", h.Admin.UpdateOrderStatus)
			r.Delete("/orders/{id}", h.Admin.DeleteOrder)

			r.Get("/knowledge", h.Admin.ListKnowledge)
			r.Post("/knowledge", h.Admin.UpsertKnowledge)
			r.Delete("/knowledge/{id}", h.Admin.DeleteKnowledge)

			r.Get("/contacts", h.Admin.ListContacts)
			r.Get("/metadata", h.Admin.GetMetadata)
			r.Get("/stats", h.Admin.GetStats)
		})
	})

	// Telegram webhook — protected by X-Telegram-Bot-Api-Secret-Token only.
	r.With(middleware.MarkTrusted).Post("/v1/telegram/webhook", h.Telegram.Webhook)

	// Internal API (X-Internal-Token)
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.InternalToken(cfg.InternalToken))
		r.Use(middleware.MarkTrusted)

		r.Post("/chat", h.Chat.Chat)
		r.Post("/chat/media", h.Chat.Media)

		// Quotes
		r.Post("/quotes", h.Quote.Create)

		// Product enrichment
		r.Post("/products/describe", h.Describe.DescribeAll)
		r.Post("/products/vectorize", h.Vectorize.VectorizeAll)
		r.Post("/products/import", h.ProductImport.Import)

		// Product images
		// i18n bulk translation
		r.Post("/products/translate", h.Translate.Translate)

		r.Post("/products/images", h.ProductImage.BulkUpload)
		r.Post("/products/images/item", h.ProductImage.ImageAdd)
		r.Put("/products/images/item", h.ProductImage.ImageUpdate)
		r.Delete("/products/images/item", h.ProductImage.ImageDelete)
	})

	return r
}
