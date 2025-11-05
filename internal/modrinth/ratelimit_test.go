package modrinth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(300, time.Minute)

	assert.NotNil(t, limiter)
	assert.Equal(t, 300, limiter.limit)
	assert.Equal(t, time.Minute, limiter.interval)
	assert.Equal(t, 300, limiter.tokens)
}

func TestRateLimiter_Wait(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		interval      time.Duration
		requestCount  int
		expectedWait  bool
		cancelContext bool
		expectedError bool
	}{
		{
			name:          "within limit",
			limit:         10,
			interval:      time.Second,
			requestCount:  5,
			expectedWait:  false,
			cancelContext: false,
			expectedError: false,
		},
		{
			name:          "exceed limit",
			limit:         2,
			interval:      100 * time.Millisecond,
			requestCount:  3,
			expectedWait:  true,
			cancelContext: false,
			expectedError: false,
		},
		{
			name:          "context cancelled",
			limit:         1,
			interval:      100 * time.Millisecond,
			requestCount:  2,
			expectedWait:  true,
			cancelContext: true,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.limit, tt.interval)

			ctx := context.Background()
			var cancel context.CancelFunc
			if tt.cancelContext {
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}

			var waitOccurred bool
			start := time.Now()

			for i := 0; i < tt.requestCount; i++ {
				// Cancel context after first request for cancel test
				if tt.cancelContext && i == 1 {
					cancel()
				}

				err := limiter.Wait(ctx)

				if tt.expectedError && err != nil {
					require.Error(t, err)
					return
				}

				if !tt.expectedError {
					require.NoError(t, err)
				}
			}

			elapsed := time.Since(start)
			if tt.expectedWait {
				waitOccurred = elapsed >= tt.interval/2
			}

			if tt.expectedWait {
				assert.True(t, waitOccurred, "expected wait to occur")
			}
		})
	}
}

func TestRateLimiter_UpdateFromHeaders(t *testing.T) {
	tests := []struct {
		name           string
		setupHeaders   func() http.Header
		expectedTokens int
		shouldUpdate   bool
	}{
		{
			name: "update remaining",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("X-RateLimit-Remaining", "150")
				return h
			},
			expectedTokens: 150,
			shouldUpdate:   true,
		},
		{
			name: "invalid remaining",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("X-RateLimit-Remaining", "invalid")
				return h
			},
			expectedTokens: 300,
			shouldUpdate:   false,
		},
		{
			name: "negative remaining",
			setupHeaders: func() http.Header {
				h := http.Header{}
				h.Set("X-RateLimit-Remaining", "-1")
				return h
			},
			expectedTokens: 300,
			shouldUpdate:   false,
		},
		{
			name: "no headers",
			setupHeaders: func() http.Header {
				return http.Header{}
			},
			expectedTokens: 300,
			shouldUpdate:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(300, time.Minute)

			headers := tt.setupHeaders()
			limiter.UpdateFromHeaders(headers)

			limiter.mu.Lock()
			actualTokens := limiter.tokens
			limiter.mu.Unlock()

			assert.Equal(t, tt.expectedTokens, actualTokens)
		})
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	limiter := NewRateLimiter(5, 100*time.Millisecond)

	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < 5; i++ {
		err := limiter.Wait(ctx)
		require.NoError(t, err)
	}

	assert.Equal(t, 0, limiter.tokens)

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should be able to consume tokens again
	err := limiter.Wait(ctx)
	require.NoError(t, err)
	assert.Equal(t, 4, limiter.tokens)
}
