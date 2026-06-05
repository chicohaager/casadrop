package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"casadrop/internal/auth"
	"casadrop/internal/handlers"
	"casadrop/internal/middleware"
	"casadrop/internal/routes"
	"casadrop/internal/storage"
)

func main() {
	// Configuration from environment
	port := getEnv("PORT", "8080")
	dataDir := getEnv("DATA_DIR", "./data")
	templatesDir := getEnv("TEMPLATES_DIR", "./web/templates")
	staticDir := getEnv("STATIC_DIR", "./web/static")
	adminPassword := getEnv("ADMIN_PASSWORD", "")

	// Initialize storage
	store, err := storage.New(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize handlers
	h, err := handlers.New(store, templatesDir)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	// Initialize admin auth with persistent sessions
	adminAuth := middleware.NewAdminAuth(adminPassword, dataDir)

	// Initialize OIDC provider (nil-safe through handlers)
	oidcProvider, err := auth.NewProvider(dataDir)
	if err != nil {
		log.Printf("OIDC: Failed to initialize provider: %v", err)
		oidcProvider = nil
	} else if oidcProvider != nil && oidcProvider.IsEnabled() {
		adminAuth.SetOIDCStatus(true, oidcProvider.IsLocalAuthDisabled())
		log.Printf("OIDC: Provider enabled")
	}

	oidcHandlers := auth.NewHandlers(oidcProvider, adminAuth)
	oidcHandlers.SetUserService(auth.NewUserService(store))

	adminAuth.SetAPIKeyValidator(store)
	// Enable per-user local (email+password) login against the users table,
	// in addition to the single admin password.
	adminAuth.SetLocalUserStore(store)

	// Initialize Email handler and start background expiry notifier
	emailHandler := handlers.NewEmailHandler(store)
	h.SetEmailHandler(emailHandler)
	emailHandler.StartExpiryNotifier()

	// Rate limiter for public download endpoint
	downloadLimiter := middleware.NewRateLimiter(10, time.Minute)

	// Build router (all URL wiring lives in internal/routes)
	router := routes.New(routes.Deps{
		Handler:         h,
		AdminAuth:       adminAuth,
		OIDC:            oidcHandlers,
		EmailHandler:    emailHandler,
		DownloadLimiter: downloadLimiter,
		StaticDir:       staticDir,
	})

	// Initialize Tailscale if configured
	handlers.InitTailscaleOnStartup(dataDir)

	logStartupBanner(port, adminAuth, oidcProvider)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown. We stop background workers AFTER srv.Shutdown()
	// returns so in-flight HTTP requests can still touch rate-limiters,
	// sessions, OIDC state, etc. while they drain.
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("Stopping background workers...")
	adminAuth.Stop()
	downloadLimiter.Stop()
	emailHandler.Stop()
	h.Stop()
	handlers.StopChunkCleanupWorker()
	if oidcProvider != nil {
		oidcProvider.Stop()
	}

	if err := store.Close(); err != nil {
		log.Printf("Storage close error: %v", err)
	}
	log.Println("Server stopped")
}

func logStartupBanner(port string, adminAuth *middleware.AdminAuth, oidcProvider *auth.Provider) {
	log.Printf("CasaDrop starting on port %s", port)
	switch {
	case adminAuth.IsEnabled():
		log.Printf("Admin authentication: ENABLED")
	case adminAuth.NeedsSetup():
		log.Printf("Admin authentication: SETUP REQUIRED (visit /setup)")
	default:
		log.Printf("Admin authentication: DISABLED")
	}
	if oidcProvider != nil && oidcProvider.IsEnabled() {
		log.Printf("OIDC authentication: ENABLED")
		if oidcProvider.IsLocalAuthDisabled() {
			log.Printf("Local auth: DISABLED (OIDC only)")
		}
	} else {
		log.Printf("OIDC authentication: DISABLED")
	}
	log.Printf("Security headers: ENABLED")
	log.Printf("Rate limiting: ENABLED")
	log.Printf("Persistent sessions: ENABLED")
	log.Printf("Prometheus metrics: ENABLED (/api/metrics, admin-only)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
