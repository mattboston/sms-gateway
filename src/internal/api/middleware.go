package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
)

type contextKey string

const (
	contextKeyUser   contextKey = "user"
	contextKeyAPIKey contextKey = "apikey"
)

// AuthMiddleware validates JWT tokens from the Authorization header or a cookie.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractJWT(r)
			if tokenStr == "" {
				writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "missing or invalid token"})
				return
			}

			claims, err := auth.ValidateJWT(jwtSecret, tokenStr)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token"})
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// KeyMiddleware validates API keys from the X-API-Key header.
func KeyMiddleware(repo *database.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "missing API key"})
				return
			}

			apiKey, err := repo.GetAPIKeyByKey(key)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid API key"})
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyAPIKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CombinedAuthMiddleware accepts either a JWT token or an API key.
func CombinedAuthMiddleware(jwtSecret string, repo *database.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try JWT first.
			if tokenStr := extractJWT(r); tokenStr != "" {
				claims, err := auth.ValidateJWT(jwtSecret, tokenStr)
				if err == nil {
					ctx := context.WithValue(r.Context(), contextKeyUser, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Try API key.
			if key := r.Header.Get("X-API-Key"); key != "" {
				apiKey, err := repo.GetAPIKeyByKey(key)
				if err == nil {
					ctx := context.WithValue(r.Context(), contextKeyAPIKey, apiKey)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		})
	}
}

// AdminMiddleware requires the authenticated user to be an admin.
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil || !claims.IsAdmin {
			writeJSON(w, http.StatusForbidden, models.ErrorResponse{Error: "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves JWT claims from the request context.
func GetUserFromContext(ctx context.Context) *auth.JWTClaims {
	claims, _ := ctx.Value(contextKeyUser).(*auth.JWTClaims)
	return claims
}

// GetAPIKeyFromContext retrieves the API key from the request context.
func GetAPIKeyFromContext(ctx context.Context) *models.APIKey {
	key, _ := ctx.Value(contextKeyAPIKey).(*models.APIKey)
	return key
}

func extractJWT(r *http.Request) string {
	// Check Authorization header.
	if header := r.Header.Get("Authorization"); header != "" {
		if strings.HasPrefix(header, "Bearer ") {
			return strings.TrimPrefix(header, "Bearer ")
		}
	}

	// Check cookie.
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}

	return ""
}
