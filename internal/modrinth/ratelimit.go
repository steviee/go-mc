package modrinth

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	mu         sync.Mutex
	limit      int
	interval   time.Duration
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:      limit,
		interval:   interval,
		tokens:     limit,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available or context is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.interval {
		r.tokens = r.limit
		r.lastRefill = now
	}

	// If no tokens available, wait for refill
	if r.tokens <= 0 {
		waitTime := r.interval - elapsed
		r.mu.Unlock()

		select {
		case <-time.After(waitTime):
			// Wait complete, refill tokens
			r.mu.Lock()
			r.tokens = r.limit
			r.lastRefill = time.Now()
		case <-ctx.Done():
			r.mu.Lock()
			return ctx.Err()
		}
	}

	// Consume a token
	r.tokens--
	return nil
}

// UpdateFromHeaders updates rate limiter state from response headers.
func (r *RateLimiter) UpdateFromHeaders(headers http.Header) {
	// Update from X-RateLimit-Remaining if present
	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if n, err := strconv.Atoi(remaining); err == nil && n >= 0 {
			r.mu.Lock()
			r.tokens = n
			r.mu.Unlock()
		}
	}

	// Update from X-RateLimit-Reset if present
	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(timestamp, 0)
			r.mu.Lock()
			if resetTime.After(r.lastRefill) {
				r.lastRefill = resetTime.Add(-r.interval)
			}
			r.mu.Unlock()
		}
	}
}
