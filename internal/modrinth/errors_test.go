package modrinth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		description string
		statusCode  int
		want        string
	}{
		{
			name:        "error with description",
			errorMsg:    "not found",
			description: "project does not exist",
			statusCode:  404,
			want:        "not found: project does not exist (status 404)",
		},
		{
			name:        "error without description",
			errorMsg:    "internal error",
			description: "",
			statusCode:  500,
			want:        "internal error (status 500)",
		},
		{
			name:        "rate limit error",
			errorMsg:    "rate limit exceeded",
			description: "too many requests from your IP",
			statusCode:  429,
			want:        "rate limit exceeded: too many requests from your IP (status 429)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{
				ErrorMsg:    tt.errorMsg,
				Description: tt.description,
				StatusCode:  tt.statusCode,
			}
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		errorMsg    string
		description string
	}{
		{
			name:        "create not found error",
			statusCode:  404,
			errorMsg:    "not found",
			description: "project does not exist",
		},
		{
			name:        "create rate limit error",
			statusCode:  429,
			errorMsg:    "rate limit exceeded",
			description: "",
		},
		{
			name:        "create server error",
			statusCode:  500,
			errorMsg:    "internal server error",
			description: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.statusCode, tt.errorMsg, tt.description)
			require.NotNil(t, err)
			assert.Equal(t, tt.statusCode, err.StatusCode)
			assert.Equal(t, tt.errorMsg, err.ErrorMsg)
			assert.Equal(t, tt.description, err.Description)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "project not found",
			err:  ErrProjectNotFound,
			want: "project not found",
		},
		{
			name: "no compatible version",
			err:  ErrNoCompatibleVersion,
			want: "no compatible version found",
		},
		{
			name: "rate limit exceeded",
			err:  ErrRateLimitExceeded,
			want: "rate limit exceeded",
		},
		{
			name: "invalid response",
			err:  ErrInvalidResponse,
			want: "invalid API response",
		},
		{
			name: "invalid search query",
			err:  ErrInvalidSearchQuery,
			want: "invalid search query",
		},
		{
			name: "circular dependency",
			err:  ErrCircularDependency,
			want: "circular dependency detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
