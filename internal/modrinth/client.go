package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	// DefaultBaseURL is the default Modrinth API base URL.
	DefaultBaseURL = "https://api.modrinth.com/v2"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second

	// UserAgent is the user agent string sent with API requests.
	UserAgent = "go-mc/dev (https://github.com/steviee/go-mc)"
)

// Client is a Modrinth API client.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	userAgent   string
	rateLimiter *RateLimiter
}

// Config holds client configuration.
type Config struct {
	BaseURL   string
	Timeout   time.Duration
	UserAgent string
}

// NewClient creates a new Modrinth API client.
func NewClient(config *Config) *Client {
	if config == nil {
		config = &Config{}
	}

	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.UserAgent == "" {
		config.UserAgent = UserAgent
	}

	slog.Debug("creating Modrinth API client",
		"base_url", config.BaseURL,
		"timeout", config.Timeout)

	return &Client{
		baseURL:     config.BaseURL,
		httpClient:  &http.Client{Timeout: config.Timeout},
		userAgent:   config.UserAgent,
		rateLimiter: NewRateLimiter(300, time.Minute), // 300 req/min
	}
}

// doRequest performs an HTTP request with rate limiting and error handling.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Build full URL
	url := c.baseURL + path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	slog.Debug("modrinth API request",
		"method", method,
		"url", url)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	// Update rate limiter from response headers
	c.rateLimiter.UpdateFromHeaders(resp.Header)

	return resp, nil
}

// parseErrorResponse parses an error response from the API.
func parseErrorResponse(resp *http.Response) error {
	defer func() {
		_ = resp.Body.Close()
	}()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimitExceeded
	}

	// Handle not found
	if resp.StatusCode == http.StatusNotFound {
		return ErrProjectNotFound
	}

	// Try to parse API error
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		// If we can't parse the error, return generic error
		return NewAPIError(resp.StatusCode, "API error", resp.Status)
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}

// checkResponse checks if the response is successful.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return parseErrorResponse(resp)
}
