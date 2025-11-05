package modrinth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetProject(t *testing.T) {
	tests := []struct {
		name           string
		idOrSlug       string
		serverResponse ProjectDetails
		expectedError  bool
	}{
		{
			name:     "get by slug",
			idOrSlug: "fabric-api",
			serverResponse: ProjectDetails{
				ID:          "P7dR8mSH",
				Slug:        "fabric-api",
				Title:       "Fabric API",
				Description: "Essential hooks for modding",
				Body:        "Full description here",
				Categories:  []string{"library", "fabric"},
				Versions:    []string{"version1", "version2"},
				Downloads:   1000000,
				IconURL:     "https://example.com/icon.png",
			},
			expectedError: false,
		},
		{
			name:     "get by ID",
			idOrSlug: "P7dR8mSH",
			serverResponse: ProjectDetails{
				ID:          "P7dR8mSH",
				Slug:        "fabric-api",
				Title:       "Fabric API",
				Description: "Essential hooks",
				Categories:  []string{"library"},
				Versions:    []string{"v1"},
				Downloads:   500,
			},
			expectedError: false,
		},
		{
			name:          "empty ID",
			idOrSlug:      "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/project/"+tt.idOrSlug, r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			project, err := client.GetProject(context.Background(), tt.idOrSlug)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.serverResponse.ID, project.ID)
			assert.Equal(t, tt.serverResponse.Slug, project.Slug)
			assert.Equal(t, tt.serverResponse.Title, project.Title)
			assert.Equal(t, tt.serverResponse.Description, project.Description)
			assert.Equal(t, len(tt.serverResponse.Versions), len(project.Versions))
			assert.Equal(t, tt.serverResponse.Downloads, project.Downloads)
		})
	}
}

func TestClient_GetProject_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":       "not_found",
			"description": "Project not found",
		})
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})

	_, err := client.GetProject(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestClient_GetProject_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})

	_, err := client.GetProject(context.Background(), "test")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestClient_GetProject_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":       "internal_error",
			"description": "Internal server error",
		})
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})

	_, err := client.GetProject(context.Background(), "test")

	require.Error(t, err)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}
