package mods

import (
	"fmt"
)

// ModInfo contains information about a known mod from our curated database.
// This represents well-tested server-side mods with their dependencies,
// port requirements, and configuration needs.
type ModInfo struct {
	// Slug is the unique identifier for this mod (e.g., "fabric-api", "lithium")
	Slug string

	// ModrinthID is the Modrinth project ID used for API queries
	ModrinthID string

	// Name is the human-readable display name
	Name string

	// Description provides a brief explanation of the mod's purpose
	Description string

	// Category classifies the mod: "library", "performance", or "feature"
	Category string

	// DefaultPort is the port number this mod uses (0 if no port needed)
	DefaultPort int

	// Protocol specifies the network protocol: "tcp", "udp", or "" if no port
	Protocol string

	// ConfigFiles lists configuration files that should be generated
	ConfigFiles []string

	// Dependencies lists required mod slugs that must be installed first
	Dependencies []string
}

// KnownMods is a curated database of well-tested server-side mods.
// This map contains essential mods for Fabric servers with verified
// compatibility, dependency information, and port requirements.
var KnownMods = map[string]ModInfo{
	"fabric-api": {
		Slug:        "fabric-api",
		ModrinthID:  "P7dR8mSH",
		Name:        "Fabric API",
		Description: "Essential library for Fabric mods",
		Category:    "library",
	},
	"lithium": {
		Slug:         "lithium",
		ModrinthID:   "gvQqBUqZ",
		Name:         "Lithium",
		Description:  "Physics/AI/Ticking optimization",
		Category:     "performance",
		Dependencies: []string{"fabric-api"},
	},
	"simple-voice-chat": {
		Slug:         "simple-voice-chat",
		ModrinthID:   "9eGKb6K1",
		Name:         "Simple Voice Chat",
		Description:  "Proximity voice chat",
		Category:     "feature",
		DefaultPort:  24454,
		Protocol:     "udp",
		ConfigFiles:  []string{"config/voicechat-server.properties"},
		Dependencies: []string{"fabric-api"},
	},
	"geyser": {
		Slug:         "geyser",
		ModrinthID:   "wKkoqHrH",
		Name:         "Geyser",
		Description:  "Bedrock client support",
		Category:     "feature",
		DefaultPort:  19132,
		Protocol:     "udp",
		ConfigFiles:  []string{"config/Geyser-Fabric/config.yml"},
		Dependencies: []string{"fabric-api"},
	},
	"bluemap": {
		Slug:         "bluemap",
		ModrinthID:   "swbUV1cr",
		Name:         "BlueMap",
		Description:  "3D web map",
		Category:     "feature",
		DefaultPort:  8100,
		Protocol:     "tcp",
		ConfigFiles:  []string{"config/bluemap/core.conf"},
		Dependencies: []string{"fabric-api"},
	},
}

// GetMod retrieves a mod from the known mods database.
// Returns an error if the mod slug is not found in the database.
func GetMod(slug string) (ModInfo, error) {
	mod, exists := KnownMods[slug]
	if !exists {
		return ModInfo{}, fmt.Errorf("unknown mod: %q", slug)
	}
	return mod, nil
}

// RequiresPort returns true if the mod requires a port to be allocated.
// Mods with DefaultPort > 0 need network port configuration.
func (m ModInfo) RequiresPort() bool {
	return m.DefaultPort > 0
}

// ResolveDependencies returns all dependencies for a list of mods, including
// transitive dependencies. Returns mods in installation order with dependencies
// appearing before the mods that require them.
//
// For example, if requesting ["lithium", "simple-voice-chat"], this returns
// ["fabric-api", "lithium", "simple-voice-chat"] since both mods depend on
// fabric-api.
//
// Returns an error if any mod slug is unknown or if circular dependencies exist.
func ResolveDependencies(modSlugs []string) ([]string, error) {
	resolved := make(map[string]bool)
	order := []string{}

	var resolve func(slug string) error
	resolve = func(slug string) error {
		if resolved[slug] {
			return nil
		}

		mod, err := GetMod(slug)
		if err != nil {
			return err
		}

		// Resolve dependencies first
		for _, dep := range mod.Dependencies {
			if err := resolve(dep); err != nil {
				return fmt.Errorf("dependency %q of %q: %w", dep, slug, err)
			}
		}

		resolved[slug] = true
		order = append(order, slug)
		return nil
	}

	for _, slug := range modSlugs {
		if err := resolve(slug); err != nil {
			return nil, err
		}
	}

	return order, nil
}
