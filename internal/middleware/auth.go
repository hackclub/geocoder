package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/models"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

func APIKeyAuth(db database.DatabaseInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from query parameter
			apiKey := r.URL.Query().Get("key")
			if apiKey == "" {
				writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "API key is required")
				return
			}

			// Hash the API key for database lookup
			keyHash := database.HashAPIKey(apiKey)

			// Look up the API key in the database
			apiKeyRecord, err := db.GetAPIKeyByHash(keyHash)
			if err != nil {
				writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "The provided API key is invalid or expired")
				return
			}

			if !apiKeyRecord.IsActive {
				writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "The provided API key has been deactivated")
				return
			}

			// Update API key usage
			_ = db.UpdateAPIKeyUsage(apiKeyRecord.ID)

			// Add API key to context
			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKeyRecord)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func BasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != username || pass != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Admin Interface"`)
				writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:      errorCode,
			Message:   message,
			Timestamp: time.Now(),
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}
