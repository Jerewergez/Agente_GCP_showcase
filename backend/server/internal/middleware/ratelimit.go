// Package middleware provides HTTP middleware components including
// rate limiting and CORS support.
package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter keyed by user ID
// or IP address.
type TokenBucket struct {
	mu       sync.Mutex
	tokens   map[string]*bucket
	rate     int           // tokens per interval
	interval time.Duration // refill interval
	burst    int           // max burst
}

type bucket struct {
	tokens    int
	lastRefill time.Time
}

// NewTokenBucket creates a rate limiter that allows `rate` operations
// per `interval`, with a maximum burst of `burst`.
func NewTokenBucket(rate int, interval time.Duration, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:   make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
}

// Allow checks whether a request from the given key is allowed.
// If not, the caller should return HTTP 429 Too Many Requests.
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	b, ok := tb.tokens[key]
	if !ok {
		b = &bucket{tokens: tb.burst, lastRefill: time.Now()}
		tb.tokens[key] = b
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	refillTokens := int(elapsed / tb.interval * time.Duration(tb.rate))
	if refillTokens > 0 {
		b.tokens += refillTokens
		if b.tokens > tb.burst {
			b.tokens = tb.burst
		}
		b.lastRefill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware returns an HTTP middleware that rate-limits
// requests per user (extracted from X-User-ID header or remote addr).
// It enforces 100 requests per minute per user with a burst of 10.
func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := NewTokenBucket(100, time.Minute, 10)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use X-User-ID header if present, otherwise remote address
		key := r.Header.Get("X-User-ID")
		if key == "" {
			key = r.RemoteAddr
		}

		if !limiter.Allow(key) {
			log.Printf("rate-limit: exceeded for %s", key)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded, try again in 60 seconds"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
