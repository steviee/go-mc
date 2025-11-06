package mojang

import "fmt"

// Error types for Mojang API operations.
var (
	ErrUsernameNotFound  = fmt.Errorf("username not found")
	ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")
	ErrInvalidUsername   = fmt.Errorf("invalid username")
	ErrAPIUnavailable    = fmt.Errorf("mojang API unavailable")
)

// APIError represents an API error with status code.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("mojang API error (status %d): %s", e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}
