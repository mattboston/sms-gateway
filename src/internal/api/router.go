package api

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	smsgateway "github.com/mattboston/sms-gateway"
	"github.com/mattboston/sms-gateway/internal/config"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/modem"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/mattboston/sms-gateway/docs" // register swagger docs
)

// NewRouter creates and configures the chi router with all middleware and routes.
func NewRouter(repo *database.Repository, m modem.Modem, cfg *config.Config) chi.Router {
	r := chi.NewRouter()

	// Standard middleware.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// CORS for dev mode.
	if cfg.DevMode {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Handlers.
	authHandler := NewAuthHandler(repo, cfg.JWTSecret)
	smsHandler := NewSMSHandler(repo, m)
	apiKeyHandler := NewKeyHandler(repo)
	modemHandler := NewModemHandler(m)
	userHandler := NewUserHandler(repo)
	healthHandler := NewHealthHandler(repo, m)

	// Auth middleware shortcuts.
	jwtAuth := AuthMiddleware(cfg.JWTSecret)
	combinedAuth := CombinedAuthMiddleware(cfg.JWTSecret, repo)

	// Public routes.
	r.Get("/api/v1/health", healthHandler.HandleHealth)
	r.Post("/api/v1/auth/login", authHandler.HandleLogin)

	// Routes requiring any authentication (JWT or API key).
	r.Group(func(r chi.Router) {
		r.Use(combinedAuth)

		r.Get("/api/v1/sms/inbox", smsHandler.HandleGetInbox)
		r.Get("/api/v1/sms/outbox", smsHandler.HandleGetOutbox)
		r.Get("/api/v1/sms/{id}", smsHandler.HandleGetMessage)
		r.Put("/api/v1/sms/{id}/read", smsHandler.HandleMarkRead)
		r.Put("/api/v1/sms/{id}/unread", smsHandler.HandleMarkUnread)
		r.Delete("/api/v1/sms/{id}", smsHandler.HandleDeleteMessage)
		r.Post("/api/v1/sms/send", smsHandler.HandleSendSMS)

		r.Get("/api/v1/modem/status", modemHandler.HandleModemStatus)
		r.Get("/api/v1/modem/signal", modemHandler.HandleModemSignal)
	})

	// Routes requiring JWT authentication.
	r.Group(func(r chi.Router) {
		r.Use(jwtAuth)

		r.Post("/api/v1/auth/logout", authHandler.HandleLogout)
		r.Post("/api/v1/auth/change-password", authHandler.HandleChangePassword)
		r.Get("/api/v1/apikeys", apiKeyHandler.HandleListAPIKeys)
		r.Post("/api/v1/apikeys", apiKeyHandler.HandleCreateAPIKey)
		r.Delete("/api/v1/apikeys/{id}", apiKeyHandler.HandleDeactivateAPIKey)
		r.Delete("/api/v1/apikeys/{id}/delete", apiKeyHandler.HandleDeleteAPIKey)
	})

	// Admin-only routes.
	r.Group(func(r chi.Router) {
		r.Use(jwtAuth)
		r.Use(AdminMiddleware)

		r.Post("/api/v1/modem/at", modemHandler.HandleSendATCommand)
		r.Get("/api/v1/users", userHandler.HandleListUsers)
		r.Post("/api/v1/users", userHandler.HandleCreateUser)
	})

	// Swagger UI.
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Serve embedded web UI for non-API routes (SPA fallback).
	webFS, err := fs.Sub(smsgateway.WebFS, "web/dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(webFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// If the request is for an API route, return 404.
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}

			// Try to serve the static file. If it doesn't exist, serve index.html (SPA fallback).
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			if _, err := fs.Stat(webFS, path); err != nil {
				// File not found, serve index.html for SPA routing.
				r.URL.Path = "/"
			}

			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
