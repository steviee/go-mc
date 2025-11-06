package mojang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the default Mojang API base URL.
	DefaultBaseURL = "https://api.mojang.com"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 10 * time.Second

	// UserAgent is the user agent string sent with API requests.
	UserAgent = "go-mc/dev (https://github.com/steviee/go-mc)"

	// RateLimitDelay is the delay between retries when rate limited.
	RateLimitDelay = 2 * time.Second

	// MaxRetries is the maximum number of retries for failed requests.
	MaxRetries = 3
)

// Client is a Mojang API client for UUID lookups.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	cache      *Cache
}

// Config holds client configuration.
type Config struct {
	BaseURL      string
	Timeout      time.Duration
	UserAgent    string
	CacheSize    int
	CacheTTL     time.Duration
	DisableCache bool
}

// NewClient creates a new Mojang API client.
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

	var cache *Cache
	if !config.DisableCache {
		cache = NewCache(config.CacheSize, config.CacheTTL)
	}

	slog.Debug("creating Mojang API client",
		"base_url", config.BaseURL,
		"timeout", config.Timeout,
		"cache_enabled", !config.DisableCache)

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: &http.Client{Timeout: config.Timeout},
		userAgent:  config.UserAgent,
		cache:      cache,
	}
}

// GetUUID looks up a UUID for the given username.
// It returns the profile or an error if the lookup fails.
// Results are cached to avoid repeated API calls.
func (c *Client) GetUUID(ctx context.Context, username string) (*Profile, error) {
	// Validate username
	if err := validateUsername(username); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidUsername, err)
	}

	// Check cache first
	if c.cache != nil {
		if entry := c.cache.Get(username); entry != nil {
			slog.Debug("mojang UUID cache hit", "username", username)
			if entry.NotFound {
				return nil, ErrUsernameNotFound
			}
			return entry.Profile, nil
		}
	}

	slog.Debug("mojang UUID cache miss, querying API", "username", username)

	// Query API with retries
	profile, err := c.queryAPIWithRetries(ctx, username)

	// Cache the result
	if c.cache != nil {
		switch err {
		case ErrUsernameNotFound:
			// Cache negative results to avoid repeated lookups
			c.cache.Set(username, CacheEntry{
				Profile:  nil,
				NotFound: true,
			})
		case nil:
			// Cache successful lookup
			c.cache.Set(username, CacheEntry{
				Profile:  profile,
				NotFound: false,
			})
		}
	}

	return profile, err
}

// queryAPIWithRetries queries the Mojang API with exponential backoff.
func (c *Client) queryAPIWithRetries(ctx context.Context, username string) (*Profile, error) {
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(attempt) * RateLimitDelay
			slog.Debug("retrying mojang API request",
				"username", username,
				"attempt", attempt+1,
				"delay", delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		profile, err := c.queryAPI(ctx, username)
		if err == nil {
			return profile, nil
		}

		lastErr = err

		// Don't retry for certain errors
		if err == ErrUsernameNotFound || err == ErrInvalidUsername {
			return nil, err
		}

		// Retry on rate limit or network errors
		if err == ErrRateLimitExceeded {
			continue
		}

		// For other errors, check if context is done
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", MaxRetries, lastErr)
}

// queryAPI performs a single API query for UUID lookup.
func (c *Client) queryAPI(ctx context.Context, username string) (*Profile, error) {
	// Build URL
	url := fmt.Sprintf("%s/users/profiles/minecraft/%s", c.baseURL, username)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	slog.Debug("mojang API request", "url", url)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIUnavailable, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Handle response
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - parse response
		var profileResp ProfileResponse
		if err := json.NewDecoder(resp.Body).Decode(&profileResp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		// Format UUID with dashes
		uuid := formatUUID(profileResp.ID)

		profile := &Profile{
			UUID:     uuid,
			Username: profileResp.Name,
		}

		slog.Debug("mojang UUID lookup success",
			"username", username,
			"uuid", uuid)

		return profile, nil

	case http.StatusNoContent, http.StatusNotFound:
		// Username not found
		slog.Debug("mojang username not found", "username", username)
		return nil, ErrUsernameNotFound

	case http.StatusTooManyRequests:
		// Rate limited
		slog.Warn("mojang API rate limit exceeded")
		return nil, ErrRateLimitExceeded

	default:
		// Other error
		body, _ := io.ReadAll(resp.Body)
		return nil, NewAPIError(resp.StatusCode, string(body))
	}
}

// validateUsername validates a Minecraft username.
// Rules:
// - Must be 1-16 characters long
// - Must contain only alphanumeric characters and underscores
func validateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) > 16 {
		return fmt.Errorf("username must be 16 characters or less, got %d", len(username))
	}

	// Minecraft usernames can only contain alphanumeric and underscores
	for _, ch := range username {
		isAlpha := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
		isDigit := ch >= '0' && ch <= '9'
		isUnderscore := ch == '_'

		if !isAlpha && !isDigit && !isUnderscore {
			return fmt.Errorf("username must contain only alphanumeric characters and underscores: %q", username)
		}
	}

	return nil
}

// formatUUID formats a UUID string with dashes.
// Input:  "069a79f444e94726a5befca90e38aaf5"
// Output: "069a79f4-44e9-4726-a5be-fca90e38aaf5"
func formatUUID(uuid string) string {
	if len(uuid) != 32 {
		return uuid
	}

	uuid = strings.ToLower(uuid)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		uuid[0:8],
		uuid[8:12],
		uuid[12:16],
		uuid[16:20],
		uuid[20:32],
	)
}

// ClearCache clears the UUID lookup cache.
func (c *Client) ClearCache() {
	if c.cache != nil {
		c.cache.Clear()
	}
}

// CacheSize returns the current number of entries in the cache.
func (c *Client) CacheSize() int {
	if c.cache == nil {
		return 0
	}
	return c.cache.Len()
}
