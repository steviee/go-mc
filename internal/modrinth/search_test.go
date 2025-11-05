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

func TestClient_Search(t *testing.T) {
	tests := []struct {
		name           string
		opts           *SearchOptions
		serverResponse SearchResult
		expectedError  bool
		validateQuery  func(*testing.T, string)
	}{
		{
			name: "basic search",
			opts: &SearchOptions{
				Query: "fabric-api",
			},
			serverResponse: SearchResult{
				Hits: []Project{
					{
						Slug:        "fabric-api",
						Title:       "Fabric API",
						Description: "Essential hooks for modding",
						ProjectID:   "P7dR8mSH",
					},
				},
				TotalHits: 1,
				Limit:     20,
				Offset:    0,
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "query=fabric-api")
			},
		},
		{
			name: "search with facets",
			opts: &SearchOptions{
				Query: "optimization",
				Facets: [][]string{
					{"project_type:mod"},
					{"categories:fabric"},
				},
				Limit: 10,
			},
			serverResponse: SearchResult{
				Hits: []Project{
					{
						Slug:  "sodium",
						Title: "Sodium",
					},
					{
						Slug:  "lithium",
						Title: "Lithium",
					},
				},
				TotalHits: 2,
				Limit:     10,
				Offset:    0,
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "query=optimization")
				assert.Contains(t, query, "facets=")
				assert.Contains(t, query, "limit=10")
			},
		},
		{
			name: "nil options uses defaults",
			opts: nil,
			serverResponse: SearchResult{
				Hits:      []Project{},
				TotalHits: 0,
				Limit:     20,
				Offset:    0,
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "limit=20")
				assert.Contains(t, query, "offset=0")
			},
		},
		{
			name: "limit over maximum is capped",
			opts: &SearchOptions{
				Query: "test",
				Limit: 150, // Over max of 100
			},
			serverResponse: SearchResult{
				Hits:      []Project{},
				TotalHits: 0,
				Limit:     100,
				Offset:    0,
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "limit=100")
			},
		},
		{
			name: "search with offset",
			opts: &SearchOptions{
				Query:  "fabric",
				Limit:  20,
				Offset: 40,
			},
			serverResponse: SearchResult{
				Hits:      []Project{},
				TotalHits: 100,
				Limit:     20,
				Offset:    40,
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "offset=40")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/search", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				if tt.validateQuery != nil {
					tt.validateQuery(t, r.URL.RawQuery)
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			result, err := client.Search(context.Background(), tt.opts)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(tt.serverResponse.Hits), len(result.Hits))
			assert.Equal(t, tt.serverResponse.TotalHits, result.TotalHits)
			assert.Equal(t, tt.serverResponse.Limit, result.Limit)
			assert.Equal(t, tt.serverResponse.Offset, result.Offset)
		})
	}
}

func TestClient_Search_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError error
	}{
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error":"not_found"}`,
			expectedError: ErrProjectNotFound,
		},
		{
			name:          "rate limit",
			statusCode:    http.StatusTooManyRequests,
			responseBody:  `{"error":"rate_limited"}`,
			expectedError: ErrRateLimitExceeded,
		},
		{
			name:          "bad request",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error":"bad_request"}`,
			expectedError: nil, // Should be APIError
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			_, err := client.Search(context.Background(), &SearchOptions{Query: "test"})

			require.Error(t, err)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestClient_SearchMods(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		limit         int
		expectedError bool
		validateQuery func(*testing.T, string)
	}{
		{
			name:          "valid search",
			query:         "fabric-api",
			limit:         10,
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "query=fabric-api")
				assert.Contains(t, query, "limit=10")
				assert.Contains(t, query, "project_type%3Amod")
				assert.Contains(t, query, "categories%3Afabric")
			},
		},
		{
			name:          "empty query",
			query:         "",
			limit:         10,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateQuery != nil {
					tt.validateQuery(t, r.URL.RawQuery)
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(SearchResult{
					Hits:      []Project{},
					TotalHits: 0,
				})
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			_, err := client.SearchMods(context.Background(), tt.query, tt.limit)

			if tt.expectedError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidSearchQuery)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClient_Search_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})

	_, err := client.Search(context.Background(), &SearchOptions{Query: "test"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}
