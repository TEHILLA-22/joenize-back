package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/handlers"
	"github.com/tehilla-22/b2b-api/internal/middleware"
	"github.com/tehilla-22/b2b-api/internal/services"
)

func New(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.CORS(cfg.FrontendURL))

	authService := services.NewAuthService(cfg)
	emailService := services.NewEmailService(cfg)
	paystackService := services.NewPaystackService(cfg)

	authHandler := handlers.NewAuthHandler(cfg, authService, emailService)
	userHandler := handlers.NewUserHandler(cfg, authService)
	orgHandler := handlers.NewOrganizationHandler(cfg)
	productHandler := handlers.NewProductHandler()
	orderHandler := handlers.NewOrderHandler()
	paymentHandler := handlers.NewPaymentHandler(cfg, paystackService)
	procurementHandler := handlers.NewProcurementHandler()
	shippingHandler := handlers.NewShippingHandler()
	notifHandler := handlers.NewNotificationHandler()
	sellerHandler := handlers.NewSellerHandler(cfg, paystackService, emailService, notifHandler)

	auth := middleware.Auth(cfg.JWTSecret)
	sellerOnly := middleware.SellerOnly

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login/", authHandler.Login)
			r.Post("/register/", authHandler.Register)
			r.Post("/verify-email/", authHandler.VerifyEmail)
			r.Post("/logout/", authHandler.Logout)
			r.Post("/refresh/", authHandler.Refresh)
			r.Post("/google/", authHandler.GoogleLogin)
			r.With(auth).Get("/me/", authHandler.Me)
		})

		r.Route("/accounts", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Patch("/users/me/", userHandler.UpdateMe)
			})
		})

		r.Route("/organizations", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/my-organization/", orgHandler.GetMyOrganization)
				r.Patch("/my-organization/", orgHandler.UpdateMyOrganization)
			})
		})

		r.Route("/products", func(r chi.Router) {
			r.Get("/", productHandler.List)
			r.Get("/featured/", productHandler.ListFeatured)
			r.Get("/categories/", productHandler.ListCategories)
			r.Get("/{id}", productHandler.Get)

			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Use(sellerOnly)
				r.Post("/", productHandler.Create)
				r.Put("/{id}", productHandler.Update)
				r.Delete("/{id}", productHandler.Delete)
				r.Get("/my/list", productHandler.ListMyProducts)
				r.Post("/{id}/images", productHandler.UploadImage)
				r.Delete("/{id}/images/{imageId}", productHandler.DeleteImage)
			})
		})

		r.Route("/orders", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/", orderHandler.List)
				r.Post("/", orderHandler.Create)
				r.Get("/{id}", orderHandler.Get)
				r.Patch("/{id}/status", orderHandler.UpdateStatus)
			})
		})

		r.Route("/procurement", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/rfqs/", procurementHandler.ListRFQs)
				r.Post("/rfqs/", procurementHandler.CreateRFQ)
				r.Post("/quotes/", procurementHandler.CreateQuote)
				r.Patch("/quotes/{id}/respond", procurementHandler.RespondToQuote)
				r.Post("/cart/", procurementHandler.AddToCart)
				r.Get("/cart/", procurementHandler.ListCart)
				r.Delete("/cart/", procurementHandler.RemoveFromCart)
			})
		})

		r.Route("/payments", func(r chi.Router) {
			r.Post("/webhook/", paymentHandler.PaystackWebhook)

			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/wallet/", paymentHandler.GetWallet)
				r.Post("/initialize/", paymentHandler.InitializePayment)
				r.Get("/verify/", paymentHandler.VerifyPayment)
			})
		})

		r.Route("/invoices", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/", orderHandler.ListInvoices)
			})
		})

		r.Route("/shipping", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Get("/", shippingHandler.List)
				r.Post("/", shippingHandler.Create)
				r.Patch("/{id}/tracking", shippingHandler.UpdateTracking)
			})
		})

		r.Route("/notifications", func(r chi.Router) {
			r.With(auth).Get("/", notifHandler.List)
			r.With(auth).Get("/stream/", notifHandler.Stream)
		})

		r.Route("/seller", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth)
				r.Patch("/profile", sellerHandler.UpdateProfile)
				r.Post("/onboarding/initialize", sellerHandler.InitializeOnboarding)
				r.Get("/onboarding/verify", sellerHandler.VerifyOnboarding)
				r.Group(func(r chi.Router) {
					r.Use(sellerOnly)
					r.Get("/dashboard", sellerHandler.DashboardSummary)
					r.Get("/escrow", sellerHandler.EscrowStatus)
				})
			})
		})
	})

	return r
}
