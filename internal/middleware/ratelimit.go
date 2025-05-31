package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/hackclub/geocoder/internal/models"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) getLimiter(apiKeyID string, rateLimitPerSecond int) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[apiKeyID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check in case another goroutine created it
		if limiter, exists = rl.limiters[apiKeyID]; !exists {
			limiter = rate.NewLimiter(rate.Limit(rateLimitPerSecond), rateLimitPerSecond)
			rl.limiters[apiKeyID] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

func (rl *RateLimiter) RateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from context
			apiKey, ok := r.Context().Value(APIKeyContextKey).(*models.APIKey)
			if !ok {
				// No API key in context, skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			limiter := rl.getLimiter(apiKey.ID, apiKey.RateLimitPerSecond)

			if !limiter.Allow() {
				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(apiKey.RateLimitPerSecond))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

				writeErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests")
				return
			}

			// Set rate limit headers for successful requests
			remaining := int(limiter.Tokens())
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(apiKey.RateLimitPerSecond))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

			next.ServeHTTP(w, r)
		})
	}
}

// Clean up old limiters periodically
func (rl *RateLimiter) Cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Remove limiters that haven't been used recently
			// This is a simple implementation; in production, you might want more sophisticated cleanup
			for key, limiter := range rl.limiters {
				if limiter.Tokens() == float64(limiter.Burst()) {
					delete(rl.limiters, key)
				}
			}
			rl.mu.Unlock()
		}
	}()
}
