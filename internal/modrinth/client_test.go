package modrinth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		config          *Config
		expectedURL     string
		expectedUA      string
		expectedTimeout time.Duration
	}{
		{
			name:            "nil config uses defaults",
			config:          nil,
			expectedURL:     DefaultBaseURL,
			expectedUA:      UserAgent,
			expectedTimeout: DefaultTimeout,
		},
		{
			name: "custom config",
			config: &Config{
				BaseURL:   "https://custom.api.com",
				Timeout:   10 * time.Second,
				UserAgent: "custom-agent",
			},
			expectedURL:     "https://custom.api.com",
			expectedUA:      "custom-agent",
			expectedTimeout: 10 * time.Second,
		},
		{
			name: "partial config uses defaults",
			config: &Config{
				BaseURL: "https://custom.api.com",
			},
			expectedURL:     "https://custom.api.com",
			expectedUA:      UserAgent,
			expectedTimeout: DefaultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			require.NotNil(t, client)
			assert.Equal(t, tt.expectedURL, client.baseURL)
			assert.Equal(t, tt.expectedUA, client.userAgent)
			assert.Equal(t, tt.expectedTimeout, client.httpClient.Timeout)
			assert.NotNil(t, client.rateLimiter)
		})
	}
}

func TestClient_doRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		serverStatus   int
		serverResponse string
		expectedError  bool
	}{
		{
			name:           "successful GET request",
			method:         "GET",
			path:           "/test",
			serverStatus:   http.StatusOK,
			serverResponse: `{"success":true}`,
			expectedError:  false,
		},
		{
			name:           "successful POST request",
			method:         "POST",
			path:           "/test",
			serverStatus:   http.StatusCreated,
			serverResponse: `{"created":true}`,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.method, r.Method)
				assert.Equal(t, tt.path, r.URL.Path)
				assert.Equal(t, UserAgent, r.Header.Get("User-Agent"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			resp, err := client.doRequest(context.Background(), tt.method, tt.path, nil)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.serverStatus, resp.StatusCode)
			_ = resp.Body.Close()
		})
	}
}

func TestClient_doRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.doRequest(ctx, "GET", "/test", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestParseErrorResponse(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError error
		errorContains string
	}{
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error":"not_found","description":"Project not found"}`,
			expectedError: ErrProjectNotFound,
		},
		{
			name:          "rate limit",
			statusCode:    http.StatusTooManyRequests,
			responseBody:  `{"error":"rate_limited","description":"Too many requests"}`,
			expectedError: ErrRateLimitExceeded,
		},
		{
			name:          "API error",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error":"bad_request","description":"Invalid parameters"}`,
			errorContains: "bad_request",
		},
		{
			name:          "invalid JSON",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `invalid json`,
			errorContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			resp, err := http.Get(server.URL)
			require.NoError(t, err)

			err = parseErrorResponse(resp)

			require.Error(t, err)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			}
			if tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains)
			}
		})
	}
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "200 OK",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "201 Created",
			statusCode:  http.StatusCreated,
			expectError: false,
		},
		{
			name:        "204 No Content",
			statusCode:  http.StatusNoContent,
			expectError: false,
		},
		{
			name:        "400 Bad Request",
			statusCode:  http.StatusBadRequest,
			expectError: true,
		},
		{
			name:        "404 Not Found",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "500 Internal Server Error",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"error":"test"}`))
			}))
			defer server.Close()

			resp, err := http.Get(server.URL)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			err = checkResponse(resp)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
