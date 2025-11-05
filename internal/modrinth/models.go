package modrinth

// SearchResult represents the response from the search API.
type SearchResult struct {
	Hits      []Project `json:"hits"`
	Offset    int       `json:"offset"`
	Limit     int       `json:"limit"`
	TotalHits int       `json:"total_hits"`
}

// Project represents a mod project in search results.
type Project struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ProjectID   string   `json:"project_id"`
	ProjectType string   `json:"project_type"`
	Downloads   int      `json:"downloads"`
	IconURL     string   `json:"icon_url"`
	Author      string   `json:"author"`
	Categories  []string `json:"categories"`
}

// ProjectDetails represents detailed information about a project.
type ProjectDetails struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	Categories  []string `json:"categories"`
	Versions    []string `json:"versions"`
	Downloads   int      `json:"downloads"`
	IconURL     string   `json:"icon_url"`
}

// Version represents a specific version of a mod.
type Version struct {
	ID            string       `json:"id"`
	ProjectID     string       `json:"project_id"`
	Name          string       `json:"name"`
	VersionNumber string       `json:"version_number"`
	Changelog     string       `json:"changelog"`
	Dependencies  []Dependency `json:"dependencies"`
	GameVersions  []string     `json:"game_versions"`
	Loaders       []string     `json:"loaders"`
	Files         []File       `json:"files"`
	DatePublished string       `json:"date_published"`
}

// Dependency represents a mod dependency.
type Dependency struct {
	VersionID      string `json:"version_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	FileName       string `json:"file_name,omitempty"`
	DependencyType string `json:"dependency_type"` // required, optional, incompatible
}

// File represents a downloadable file.
type File struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	Size     int64  `json:"size"`
	FileType string `json:"file_type"`
}

// VersionFilter holds version filtering criteria.
type VersionFilter struct {
	Loaders      []string // e.g., ["fabric"]
	GameVersions []string // e.g., ["1.21.1"]
}

// SearchOptions holds search parameters.
type SearchOptions struct {
	Query  string
	Facets [][]string // e.g., [["project_type:mod"], ["categories:fabric"]]
	Limit  int        // default: 20, max: 100
	Offset int
}
