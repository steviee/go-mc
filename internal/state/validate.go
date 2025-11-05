package state

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// serverNameRegex validates server names (alphanumeric + hyphen only)
	serverNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)

	// uuidRegex validates UUID format
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// memoryRegex validates memory format (e.g., "2G", "512M", "4096M")
	memoryRegex = regexp.MustCompile(`^[0-9]+[MGT]$`)
)

// ValidateServerName validates a server name.
// Rules:
// - Must be 1-63 characters long
// - Must contain only alphanumeric characters and hyphens
// - Must start and end with alphanumeric character
// - Must not be empty
func ValidateServerName(name string) error {
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("server name must be 63 characters or less, got %d", len(name))
	}

	if !serverNameRegex.MatchString(name) {
		return fmt.Errorf("server name must contain only alphanumeric characters and hyphens, and start/end with alphanumeric: %q", name)
	}

	return nil
}

// ValidateUUID validates a UUID string.
func ValidateUUID(uuid string) error {
	if uuid == "" {
		return fmt.Errorf("UUID cannot be empty")
	}

	uuid = strings.ToLower(uuid)
	if !uuidRegex.MatchString(uuid) {
		return fmt.Errorf("invalid UUID format: %q", uuid)
	}

	return nil
}

// ValidatePort validates a port number.
// Valid range: 1-65535
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidateMemory validates a memory size string.
// Valid formats: "512M", "2G", "4096M", etc.
func ValidateMemory(memory string) error {
	if memory == "" {
		return fmt.Errorf("memory cannot be empty")
	}

	if !memoryRegex.MatchString(memory) {
		return fmt.Errorf("invalid memory format: %q (expected format: 512M, 2G, etc.)", memory)
	}

	return nil
}

// ValidateVersion validates a Minecraft version string.
// This is a basic check that the version is not empty.
// More sophisticated validation could check against known versions.
func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Version must not contain spaces
	if strings.Contains(version, " ") {
		return fmt.Errorf("version cannot contain spaces: %q", version)
	}

	return nil
}

// ValidateWhitelistName validates a whitelist name.
// Same rules as server name.
func ValidateWhitelistName(name string) error {
	if name == "" {
		return fmt.Errorf("whitelist name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("whitelist name must be 63 characters or less, got %d", len(name))
	}

	if !serverNameRegex.MatchString(name) {
		return fmt.Errorf("whitelist name must contain only alphanumeric characters and hyphens, and start/end with alphanumeric: %q", name)
	}

	return nil
}

// ValidatePlayerName validates a Minecraft player name.
// Rules:
// - Must be 1-16 characters long
// - Must contain only alphanumeric characters and underscores
func ValidatePlayerName(name string) error {
	if name == "" {
		return fmt.Errorf("player name cannot be empty")
	}

	if len(name) > 16 {
		return fmt.Errorf("player name must be 16 characters or less, got %d", len(name))
	}

	// Minecraft usernames can only contain alphanumeric and underscores
	for _, ch := range name {
		isAlpha := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
		isDigit := ch >= '0' && ch <= '9'
		isUnderscore := ch == '_'

		if !isAlpha && !isDigit && !isUnderscore {
			return fmt.Errorf("player name must contain only alphanumeric characters and underscores: %q", name)
		}
	}

	return nil
}

// ValidateJavaVersion validates a Java version number.
// Valid values: 8, 11, 17, 21, etc.
func ValidateJavaVersion(version int) error {
	validVersions := []int{8, 11, 17, 21}

	for _, v := range validVersions {
		if version == v {
			return nil
		}
	}

	return fmt.Errorf("invalid Java version: %d (valid versions: 8, 11, 17, 21)", version)
}

// ValidateOpLevel validates an operator permission level.
// Valid range: 1-4
func ValidateOpLevel(level int) error {
	if level < 1 || level > 4 {
		return fmt.Errorf("op level must be between 1 and 4, got %d", level)
	}
	return nil
}

// ValidatePath validates a file path.
// This is a basic check to prevent directory traversal attacks.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Prevent directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path cannot contain '..': %q", path)
	}

	return nil
}
