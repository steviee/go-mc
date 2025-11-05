package modrinth

import (
	"errors"
	"fmt"
)

// Sentinel errors for Modrinth API operations.
var (
	// ErrProjectNotFound is returned when a project cannot be found.
	ErrProjectNotFound = errors.New("project not found")

	// ErrNoCompatibleVersion is returned when no compatible version is found.
	ErrNoCompatibleVersion = errors.New("no compatible version found")

	// ErrRateLimitExceeded is returned when the API rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidResponse is returned when the API returns an invalid response.
	ErrInvalidResponse = errors.New("invalid API response")

	// ErrInvalidSearchQuery is returned when the search query is invalid.
	ErrInvalidSearchQuery = errors.New("invalid search query")

	// ErrCircularDependency is returned when a circular dependency is detected.
	ErrCircularDependency = errors.New("circular dependency detected")
)

// APIError represents an API error response.
type APIError struct {
	ErrorMsg    string `json:"error"`
	Description string `json:"description"`
	StatusCode  int    `json:"-"`
}

// Error returns the error message.
func (e *APIError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s (status %d)", e.ErrorMsg, e.Description, e.StatusCode)
	}
	return fmt.Sprintf("%s (status %d)", e.ErrorMsg, e.StatusCode)
}

// NewAPIError creates a new APIError.
func NewAPIError(statusCode int, errorMsg, description string) *APIError {
	return &APIError{
		ErrorMsg:    errorMsg,
		Description: description,
		StatusCode:  statusCode,
	}
}
