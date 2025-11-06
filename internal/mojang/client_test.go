package mojang

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   *Client
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want: &Client{
				baseURL:   DefaultBaseURL,
				userAgent: UserAgent,
			},
		},
		{
			name: "custom config",
			config: &Config{
				BaseURL:   "https://custom.api.example.com",
				Timeout:   5 * time.Second,
				UserAgent: "custom-agent",
			},
			want: &Client{
				baseURL:   "https://custom.api.example.com",
				userAgent: "custom-agent",
			},
		},
		{
			name: "disable cache",
			config: &Config{
				DisableCache: true,
			},
			want: &Client{
				baseURL:   DefaultBaseURL,
				userAgent: UserAgent,
				cache:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient(tt.config)

			assert.Equal(t, tt.want.baseURL, got.baseURL)
			assert.Equal(t, tt.want.userAgent, got.userAgent)
			assert.NotNil(t, got.httpClient)

			if tt.config != nil && tt.config.DisableCache {
				assert.Nil(t, got.cache)
			} else {
				assert.NotNil(t, got.cache)
			}
		})
	}
}

func TestClient_GetUUID(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		statusCode int
		response   interface{}
		wantUUID   string
		wantName   string
		wantErr    error
	}{
		{
			name:       "successful lookup",
			username:   "Notch",
			statusCode: http.StatusOK,
			response: ProfileResponse{
				ID:   "069a79f444e94726a5befca90e38aaf5",
				Name: "Notch",
			},
			wantUUID: "069a79f4-44e9-4726-a5be-fca90e38aaf5",
			wantName: "Notch",
			wantErr:  nil,
		},
		{
			name:       "username not found",
			username:   "NonExistUser123",
			statusCode: http.StatusNoContent,
			response:   nil,
			wantErr:    ErrUsernameNotFound,
		},
		{
			name:       "username not found 404",
			username:   "NonExistUser123",
			statusCode: http.StatusNotFound,
			response:   nil,
			wantErr:    ErrUsernameNotFound,
		},
		{
			name:       "rate limit exceeded",
			username:   "Notch",
			statusCode: http.StatusTooManyRequests,
			response:   nil,
			wantErr:    ErrRateLimitExceeded,
		},
		{
			name:     "invalid username - empty",
			username: "",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid username - too long",
			username: "ThisUsernameIsWayTooLongForMinecraft",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid username - special chars",
			username: "User@Name",
			wantErr:  ErrInvalidUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/users/profiles/minecraft/"+tt.username, r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					err := json.NewEncoder(w).Encode(tt.response)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			// Create client with test server
			client := NewClient(&Config{
				BaseURL:      server.URL,
				Timeout:      5 * time.Second,
				DisableCache: true,
			})

			// Test GetUUID
			ctx := context.Background()
			profile, err := client.GetUUID(ctx, tt.username)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, profile)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profile)
				assert.Equal(t, tt.wantUUID, profile.UUID)
				assert.Equal(t, tt.wantName, profile.Username)
			}
		})
	}
}

func TestClient_GetUUID_Cache(t *testing.T) {
	requestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		resp := ProfileResponse{
			ID:   "069a79f444e94726a5befca90e38aaf5",
			Name: "Notch",
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	// Create client with cache
	client := NewClient(&Config{
		BaseURL:   server.URL,
		CacheSize: 10,
		CacheTTL:  1 * time.Minute,
	})

	ctx := context.Background()

	// First request - should hit API
	profile1, err := client.GetUUID(ctx, "Notch")
	require.NoError(t, err)
	require.NotNil(t, profile1)
	assert.Equal(t, 1, requestCount)
	assert.Equal(t, 1, client.CacheSize())

	// Second request - should hit cache
	profile2, err := client.GetUUID(ctx, "Notch")
	require.NoError(t, err)
	require.NotNil(t, profile2)
	assert.Equal(t, 1, requestCount, "should not make another API request")
	assert.Equal(t, profile1.UUID, profile2.UUID)

	// Case insensitive - should hit cache
	profile3, err := client.GetUUID(ctx, "notch")
	require.NoError(t, err)
	require.NotNil(t, profile3)
	assert.Equal(t, 1, requestCount, "should not make another API request")
	assert.Equal(t, profile1.UUID, profile3.UUID)
}

func TestClient_GetUUID_CacheNegative(t *testing.T) {
	requestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client with cache
	client := NewClient(&Config{
		BaseURL:  server.URL,
		CacheTTL: 1 * time.Minute,
	})

	ctx := context.Background()

	// First request - should hit API
	profile1, err := client.GetUUID(ctx, "NonExistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUsernameNotFound)
	assert.Nil(t, profile1)
	assert.Equal(t, 1, requestCount)

	// Second request - should hit cache (negative result)
	profile2, err := client.GetUUID(ctx, "NonExistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUsernameNotFound)
	assert.Nil(t, profile2)
	assert.Equal(t, 1, requestCount, "should not make another API request")
}

func TestClient_GetUUID_ContextCancel(t *testing.T) {
	// Create test server that delays
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL:      server.URL,
		DisableCache: true,
	})

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	profile, err := client.GetUUID(ctx, "Notch")
	require.Error(t, err)
	assert.Nil(t, profile)
}

func TestClient_GetUUID_Retry(t *testing.T) {
	requestCount := 0

	// Create test server that fails first time, succeeds second time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request - rate limited
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// Second request - success
		w.WriteHeader(http.StatusOK)
		resp := ProfileResponse{
			ID:   "069a79f444e94726a5befca90e38aaf5",
			Name: "Notch",
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient(&Config{
		BaseURL:      server.URL,
		DisableCache: true,
	})

	ctx := context.Background()
	profile, err := client.GetUUID(ctx, "Notch")
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, 2, requestCount, "should retry after rate limit")
}

func TestClient_ClearCache(t *testing.T) {
	client := NewClient(&Config{
		CacheSize: 10,
	})

	// Add some entries to cache
	client.cache.Set("user1", CacheEntry{
		Profile: &Profile{UUID: "uuid1", Username: "user1"},
	})
	client.cache.Set("user2", CacheEntry{
		Profile: &Profile{UUID: "uuid2", Username: "user2"},
	})

	assert.Equal(t, 2, client.CacheSize())

	// Clear cache
	client.ClearCache()

	assert.Equal(t, 0, client.CacheSize())
}

func TestFormatUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid UUID without dashes",
			input: "069a79f444e94726a5befca90e38aaf5",
			want:  "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		},
		{
			name:  "uppercase UUID",
			input: "069A79F444E94726A5BEFCA90E38AAF5",
			want:  "069a79f4-44e9-4726-a5be-fca90e38aaf5",
		},
		{
			name:  "invalid length - no change",
			input: "invalid",
			want:  "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUUID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "valid username",
			username: "Notch",
			wantErr:  false,
		},
		{
			name:     "valid username with underscore",
			username: "jeb_",
			wantErr:  false,
		},
		{
			name:     "valid username with numbers",
			username: "Player123",
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  true,
		},
		{
			name:     "too long username",
			username: "ThisUsernameIsWayTooLongForMinecraft",
			wantErr:  true,
		},
		{
			name:     "invalid character - space",
			username: "User Name",
			wantErr:  true,
		},
		{
			name:     "invalid character - special",
			username: "User@Name",
			wantErr:  true,
		},
		{
			name:     "invalid character - dash",
			username: "User-Name",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		want       string
	}{
		{
			name:       "basic error",
			statusCode: 404,
			message:    "not found",
			want:       "mojang API error (status 404): not found",
		},
		{
			name:       "rate limit error",
			statusCode: 429,
			message:    "too many requests",
			want:       "mojang API error (status 429): too many requests",
		},
		{
			name:       "server error",
			statusCode: 500,
			message:    "internal server error",
			want:       "mojang API error (status 500): internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{
				StatusCode: tt.statusCode,
				Message:    tt.message,
			}
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "create not found error",
			statusCode: 404,
			message:    "player not found",
		},
		{
			name:       "create rate limit error",
			statusCode: 429,
			message:    "rate limit exceeded",
		},
		{
			name:       "create generic error",
			statusCode: 503,
			message:    "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.statusCode, tt.message)
			require.NotNil(t, err)
			assert.Equal(t, tt.statusCode, err.StatusCode)
			assert.Equal(t, tt.message, err.Message)
			assert.Contains(t, err.Error(), tt.message)
			assert.Contains(t, err.Error(), "mojang API error")
		})
	}
}
