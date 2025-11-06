package mojang

import "time"

// ProfileResponse represents the response from the Mojang profile API.
type ProfileResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Profile represents a Minecraft player profile with UUID.
type Profile struct {
	UUID     string
	Username string
}

// CacheEntry represents a cached UUID lookup result.
type CacheEntry struct {
	Profile   *Profile
	Timestamp time.Time
	NotFound  bool // True if username doesn't exist
}
