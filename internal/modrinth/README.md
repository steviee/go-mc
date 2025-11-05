# Modrinth API Client

This package provides a Go client for the Modrinth API v2, specifically designed for searching and downloading Fabric mods with automatic dependency resolution.

## Features

- **Search**: Search for mods with filtering and pagination
- **Project Details**: Fetch detailed information about mods
- **Version Matching**: Find compatible versions for specific Minecraft and loader versions
- **Dependency Resolution**: Automatically resolve required mod dependencies
- **Rate Limiting**: Built-in rate limiting to respect API limits (300 req/min)
- **Error Handling**: Comprehensive error types for different failure scenarios
- **Context Support**: Full context propagation for cancellation and timeouts

## Usage

### Creating a Client

```go
// Use default configuration
client := modrinth.NewClient(nil)

// Or customize configuration
client := modrinth.NewClient(&modrinth.Config{
    BaseURL:   modrinth.DefaultBaseURL,
    Timeout:   30 * time.Second,
    UserAgent: "go-mc/1.0.0",
})
```

### Searching for Mods

```go
ctx := context.Background()

// Simple search for Fabric mods
results, err := client.SearchMods(ctx, "fabric-api", 10)
if err != nil {
    log.Fatal(err)
}

for _, mod := range results.Hits {
    fmt.Printf("%s: %s\n", mod.Title, mod.Description)
}
```

### Advanced Search

```go
results, err := client.Search(ctx, &modrinth.SearchOptions{
    Query: "optimization",
    Facets: [][]string{
        {"project_type:mod"},
        {"categories:fabric"},
    },
    Limit:  20,
    Offset: 0,
})
```

### Finding Compatible Versions

```go
version, err := client.FindCompatibleVersion(
    ctx,
    "P7dR8mSH",  // Project ID (Fabric API)
    "1.21.1",     // Minecraft version
    "",           // Loader version (optional)
)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Version: %s\n", version.VersionNumber)
```

### Resolving Dependencies

```go
// After finding a version, resolve its dependencies
deps, err := client.ResolveDependencies(ctx, version, "1.21.1")
if err != nil {
    log.Fatal(err)
}

for _, dep := range deps {
    fmt.Printf("Dependency: %s (%s)\n", dep.Name, dep.VersionNumber)
}
```

### Downloading Files

```go
file, err := modrinth.GetPrimaryFile(version)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Download: %s\n", file.URL)
fmt.Printf("Size: %d bytes\n", file.Size)
```

## API Structure

### Client

The main client type with methods for API operations:

- `Search(ctx, opts)` - Search for projects
- `SearchMods(ctx, query, limit)` - Convenience method for mod search
- `GetProject(ctx, idOrSlug)` - Get project details
- `GetVersions(ctx, projectID, filter)` - Get project versions
- `FindCompatibleVersion(ctx, projectID, mcVersion, loaderVersion)` - Find compatible version
- `ResolveDependencies(ctx, version, mcVersion)` - Resolve dependencies recursively

### Models

- `Project` - Mod project information
- `ProjectDetails` - Detailed project information
- `Version` - Mod version with metadata
- `Dependency` - Dependency information
- `File` - Downloadable file information

### Error Types

- `ErrProjectNotFound` - Project does not exist
- `ErrNoCompatibleVersion` - No version matches criteria
- `ErrRateLimitExceeded` - API rate limit hit
- `ErrInvalidResponse` - Invalid API response
- `ErrInvalidSearchQuery` - Invalid search query
- `ErrCircularDependency` - Circular dependency detected

## Rate Limiting

The client implements token bucket rate limiting:

- **Limit**: 300 requests per minute (Modrinth API limit)
- **Auto-Update**: Updates from `X-RateLimit-Remaining` headers
- **Context-Aware**: Respects context cancellation during rate limit waits

## Best Practices

1. **Reuse Client**: Create one client and reuse it across requests
2. **Context Timeout**: Always use context with timeout for API calls
3. **Error Handling**: Check for specific error types (e.g., `ErrProjectNotFound`)
4. **Dependency Resolution**: Be aware that dependency resolution is recursive and may make multiple API calls

## Example: Complete Workflow

```go
func installMod(ctx context.Context, modSlug, mcVersion string) error {
    client := modrinth.NewClient(nil)

    // 1. Get project details
    project, err := client.GetProject(ctx, modSlug)
    if err != nil {
        return fmt.Errorf("get project: %w", err)
    }

    // 2. Find compatible version
    version, err := client.FindCompatibleVersion(ctx, project.ID, mcVersion, "")
    if err != nil {
        return fmt.Errorf("find version: %w", err)
    }

    // 3. Resolve dependencies
    deps, err := client.ResolveDependencies(ctx, version, mcVersion)
    if err != nil {
        return fmt.Errorf("resolve deps: %w", err)
    }

    // 4. Download all files (mod + dependencies)
    allVersions := append([]modrinth.Version{*version}, deps...)
    for _, v := range allVersions {
        file, err := modrinth.GetPrimaryFile(&v)
        if err != nil {
            return fmt.Errorf("get file: %w", err)
        }

        fmt.Printf("Download: %s\n", file.URL)
        // Download file.URL to mods folder
    }

    return nil
}
```

## Testing

The package includes comprehensive tests with 88.9% coverage:

```bash
go test -v ./internal/modrinth/...
go test -race ./internal/modrinth/...
go test -cover ./internal/modrinth/...
```

## Thread Safety

The client is safe for concurrent use. The internal rate limiter uses mutexes to protect shared state.
