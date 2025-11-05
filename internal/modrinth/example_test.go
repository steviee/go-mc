package modrinth_test

import (
	"context"
	"fmt"
	"log"

	"github.com/steviee/go-mc/internal/modrinth"
)

// ExampleClient_SearchMods demonstrates searching for Fabric mods.
func ExampleClient_SearchMods() {
	// Create a new Modrinth API client
	client := modrinth.NewClient(nil) // nil uses default config

	// Search for mods
	ctx := context.Background()
	results, err := client.SearchMods(ctx, "fabric-api", 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d mods\n", len(results.Hits))
	for _, mod := range results.Hits {
		fmt.Printf("- %s: %s\n", mod.Title, mod.Description)
	}
}

// ExampleClient_GetProject demonstrates fetching project details.
func ExampleClient_GetProject() {
	client := modrinth.NewClient(nil)

	ctx := context.Background()
	project, err := client.GetProject(ctx, "fabric-api")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Project: %s\n", project.Title)
	fmt.Printf("Description: %s\n", project.Description)
	fmt.Printf("Downloads: %d\n", project.Downloads)
}

// ExampleClient_FindCompatibleVersion demonstrates finding a compatible mod version.
func ExampleClient_FindCompatibleVersion() {
	client := modrinth.NewClient(nil)

	ctx := context.Background()
	version, err := client.FindCompatibleVersion(ctx, "P7dR8mSH", "1.21.1", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Version: %s\n", version.VersionNumber)
	fmt.Printf("Loaders: %v\n", version.Loaders)
	fmt.Printf("Game Versions: %v\n", version.GameVersions)
}

// ExampleClient_ResolveDependencies demonstrates resolving mod dependencies.
func ExampleClient_ResolveDependencies() {
	client := modrinth.NewClient(nil)

	ctx := context.Background()

	// First, find a version
	version, err := client.FindCompatibleVersion(ctx, "P7dR8mSH", "1.21.1", "")
	if err != nil {
		log.Fatal(err)
	}

	// Resolve its dependencies
	deps, err := client.ResolveDependencies(ctx, version, "1.21.1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d dependencies\n", len(deps))
	for _, dep := range deps {
		fmt.Printf("- %s (%s)\n", dep.Name, dep.VersionNumber)
	}
}

// ExampleGetPrimaryFile demonstrates getting the primary file from a version.
func ExampleGetPrimaryFile() {
	version := &modrinth.Version{
		Files: []modrinth.File{
			{Filename: "mod-1.0.0.jar", Primary: false},
			{Filename: "mod-1.0.0-fabric.jar", Primary: true, URL: "https://example.com/download"},
		},
	}

	file, err := modrinth.GetPrimaryFile(version)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Primary file: %s\n", file.Filename)
	fmt.Printf("Download URL: %s\n", file.URL)
}

// Example_customConfig demonstrates creating a client with custom configuration.
func Example_customConfig() {
	config := &modrinth.Config{
		BaseURL:   modrinth.DefaultBaseURL,
		Timeout:   60 * 1000, // 60 seconds
		UserAgent: "go-mc/1.0.0",
	}

	client := modrinth.NewClient(config)

	ctx := context.Background()
	results, err := client.Search(ctx, &modrinth.SearchOptions{
		Query: "optimization",
		Facets: [][]string{
			{"project_type:mod"},
			{"categories:fabric"},
		},
		Limit: 5,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d optimization mods\n", len(results.Hits))
}
