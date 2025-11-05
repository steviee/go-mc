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

func TestClient_GetVersions(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		filter         *VersionFilter
		serverResponse []Version
		expectedError  bool
		validateQuery  func(*testing.T, string)
	}{
		{
			name:      "get all versions",
			projectID: "P7dR8mSH",
			filter:    nil,
			serverResponse: []Version{
				{
					ID:            "version1",
					ProjectID:     "P7dR8mSH",
					Name:          "1.0.0",
					VersionNumber: "1.0.0",
					GameVersions:  []string{"1.21.1"},
					Loaders:       []string{"fabric"},
				},
			},
			expectedError: false,
		},
		{
			name:      "filter by loader and game version",
			projectID: "P7dR8mSH",
			filter: &VersionFilter{
				Loaders:      []string{"fabric"},
				GameVersions: []string{"1.21.1"},
			},
			serverResponse: []Version{
				{
					ID:            "version1",
					ProjectID:     "P7dR8mSH",
					Name:          "1.0.0",
					VersionNumber: "1.0.0",
					GameVersions:  []string{"1.21.1"},
					Loaders:       []string{"fabric"},
				},
			},
			expectedError: false,
			validateQuery: func(t *testing.T, query string) {
				assert.Contains(t, query, "loaders=")
				assert.Contains(t, query, "game_versions=")
			},
		},
		{
			name:          "empty project ID",
			projectID:     "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/project/")
				assert.Contains(t, r.URL.Path, "/version")
				assert.Equal(t, "GET", r.Method)

				if tt.validateQuery != nil {
					tt.validateQuery(t, r.URL.RawQuery)
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			versions, err := client.GetVersions(context.Background(), tt.projectID, tt.filter)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(tt.serverResponse), len(versions))
			if len(versions) > 0 {
				assert.Equal(t, tt.serverResponse[0].ID, versions[0].ID)
				assert.Equal(t, tt.serverResponse[0].VersionNumber, versions[0].VersionNumber)
			}
		})
	}
}

func TestClient_FindCompatibleVersion(t *testing.T) {
	tests := []struct {
		name             string
		projectID        string
		minecraftVersion string
		loaderVersion    string
		serverResponse   []Version
		expectedError    error
	}{
		{
			name:             "find compatible version",
			projectID:        "P7dR8mSH",
			minecraftVersion: "1.21.1",
			loaderVersion:    "0.16.0",
			serverResponse: []Version{
				{
					ID:            "version1",
					ProjectID:     "P7dR8mSH",
					Name:          "1.0.0",
					VersionNumber: "1.0.0",
					GameVersions:  []string{"1.21.1"},
					Loaders:       []string{"fabric"},
				},
			},
			expectedError: nil,
		},
		{
			name:             "no compatible version",
			projectID:        "P7dR8mSH",
			minecraftVersion: "1.21.1",
			loaderVersion:    "0.16.0",
			serverResponse:   []Version{},
			expectedError:    ErrNoCompatibleVersion,
		},
		{
			name:             "empty project ID",
			projectID:        "",
			minecraftVersion: "1.21.1",
			expectedError:    nil, // Will error with different message
		},
		{
			name:             "empty minecraft version",
			projectID:        "P7dR8mSH",
			minecraftVersion: "",
			expectedError:    nil, // Will error with different message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			version, err := client.FindCompatibleVersion(context.Background(), tt.projectID, tt.minecraftVersion, tt.loaderVersion)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}

			if tt.projectID == "" || tt.minecraftVersion == "" {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.serverResponse[0].ID, version.ID)
			assert.Equal(t, tt.serverResponse[0].VersionNumber, version.VersionNumber)
		})
	}
}

func TestGetPrimaryFile(t *testing.T) {
	tests := []struct {
		name          string
		version       *Version
		expectedFile  *File
		expectedError bool
	}{
		{
			name: "primary file marked",
			version: &Version{
				Files: []File{
					{Filename: "mod-1.0.0.jar", Primary: false},
					{Filename: "mod-1.0.0-primary.jar", Primary: true},
				},
			},
			expectedFile: &File{
				Filename: "mod-1.0.0-primary.jar",
				Primary:  true,
			},
			expectedError: false,
		},
		{
			name: "no primary file, return first",
			version: &Version{
				Files: []File{
					{Filename: "mod-1.0.0.jar", Primary: false},
					{Filename: "mod-1.0.0-dev.jar", Primary: false},
				},
			},
			expectedFile: &File{
				Filename: "mod-1.0.0.jar",
				Primary:  false,
			},
			expectedError: false,
		},
		{
			name:          "nil version",
			version:       nil,
			expectedError: true,
		},
		{
			name: "no files",
			version: &Version{
				Files: []File{},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := GetPrimaryFile(tt.version)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedFile.Filename, file.Filename)
			assert.Equal(t, tt.expectedFile.Primary, file.Primary)
		})
	}
}

func TestClient_ResolveDependencies(t *testing.T) {
	tests := []struct {
		name             string
		rootVersion      *Version
		minecraftVersion string
		mockResponses    map[string][]Version // projectID -> versions
		expectedCount    int
		expectedError    bool
	}{
		{
			name: "no dependencies",
			rootVersion: &Version{
				ID:           "root",
				ProjectID:    "root-project",
				Dependencies: []Dependency{},
			},
			minecraftVersion: "1.21.1",
			expectedCount:    0,
			expectedError:    false,
		},
		{
			name: "single required dependency",
			rootVersion: &Version{
				ID:        "root",
				ProjectID: "root-project",
				Dependencies: []Dependency{
					{
						ProjectID:      "dep1",
						DependencyType: "required",
					},
				},
			},
			minecraftVersion: "1.21.1",
			mockResponses: map[string][]Version{
				"dep1": {
					{
						ID:           "dep1-version",
						ProjectID:    "dep1",
						Dependencies: []Dependency{},
					},
				},
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "nested dependencies",
			rootVersion: &Version{
				ID:        "root",
				ProjectID: "root-project",
				Dependencies: []Dependency{
					{
						ProjectID:      "dep1",
						DependencyType: "required",
					},
				},
			},
			minecraftVersion: "1.21.1",
			mockResponses: map[string][]Version{
				"dep1": {
					{
						ID:        "dep1-version",
						ProjectID: "dep1",
						Dependencies: []Dependency{
							{
								ProjectID:      "dep2",
								DependencyType: "required",
							},
						},
					},
				},
				"dep2": {
					{
						ID:           "dep2-version",
						ProjectID:    "dep2",
						Dependencies: []Dependency{},
					},
				},
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "optional dependencies are skipped",
			rootVersion: &Version{
				ID:        "root",
				ProjectID: "root-project",
				Dependencies: []Dependency{
					{
						ProjectID:      "dep1",
						DependencyType: "optional",
					},
					{
						ProjectID:      "dep2",
						DependencyType: "required",
					},
				},
			},
			minecraftVersion: "1.21.1",
			mockResponses: map[string][]Version{
				"dep2": {
					{
						ID:           "dep2-version",
						ProjectID:    "dep2",
						Dependencies: []Dependency{},
					},
				},
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "circular dependency",
			rootVersion: &Version{
				ID:        "root",
				ProjectID: "root-project",
				Dependencies: []Dependency{
					{
						ProjectID:      "dep1",
						DependencyType: "required",
					},
				},
			},
			minecraftVersion: "1.21.1",
			mockResponses: map[string][]Version{
				"dep1": {
					{
						ID:        "dep1-version",
						ProjectID: "dep1",
						Dependencies: []Dependency{
							{
								ProjectID:      "root-project",
								DependencyType: "required",
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:             "nil version",
			rootVersion:      nil,
			minecraftVersion: "1.21.1",
			expectedError:    true,
		},
		{
			name: "empty minecraft version",
			rootVersion: &Version{
				ID:        "root",
				ProjectID: "root-project",
			},
			minecraftVersion: "",
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract project ID from path
				// Path format: /project/{projectID}/version
				path := r.URL.Path
				projectID := ""
				if len(path) > 9 && path[:9] == "/project/" {
					endIdx := len(path)
					if idx := len(path) - 8; idx > 9 && path[idx:] == "/version" {
						endIdx = idx
					}
					projectID = path[9:endIdx]
				}

				if versions, ok := tt.mockResponses[projectID]; ok {
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(versions)
				} else {
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode([]Version{})
				}
			}))
			defer server.Close()

			client := NewClient(&Config{BaseURL: server.URL})

			deps, err := client.ResolveDependencies(context.Background(), tt.rootVersion, tt.minecraftVersion)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(deps))
		})
	}
}
