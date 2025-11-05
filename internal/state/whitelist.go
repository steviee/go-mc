package state

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// WhitelistState represents a whitelist with players.
type WhitelistState struct {
	Name      string       `yaml:"name"`
	CreatedAt time.Time    `yaml:"created_at"`
	UpdatedAt time.Time    `yaml:"updated_at"`
	Players   []PlayerInfo `yaml:"players"`
}

// PlayerInfo represents a whitelisted player.
type PlayerInfo struct {
	UUID    string    `yaml:"uuid"`
	Name    string    `yaml:"name"`
	AddedAt time.Time `yaml:"added_at"`
}

// NewWhitelistState creates a new WhitelistState with default values.
func NewWhitelistState(name string) *WhitelistState {
	now := time.Now()
	return &WhitelistState{
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Players:   []PlayerInfo{},
	}
}

// LoadWhitelistState loads a whitelist's state from its YAML file.
// If the file doesn't exist, it returns an error.
// If the file is corrupted, it backs up the corrupted file and returns an error.
func LoadWhitelistState(ctx context.Context, name string) (*WhitelistState, error) {
	if err := ValidateWhitelistName(name); err != nil {
		return nil, err
	}

	whitelistPath, err := GetWhitelistPath(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelist path: %w", err)
	}

	// Check if whitelist file exists
	if _, err := os.Stat(whitelistPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("whitelist %q does not exist", name)
	}

	// Read whitelist file
	data, err := os.ReadFile(whitelistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read whitelist file: %w", err)
	}

	// Parse YAML
	var state WhitelistState
	if err := yaml.Unmarshal(data, &state); err != nil {
		// Whitelist file is corrupted, backup
		backupPath := whitelistPath + ".corrupted"
		if backupErr := os.Rename(whitelistPath, backupPath); backupErr != nil {
			return nil, fmt.Errorf("whitelist file is corrupted and failed to create backup: %w (original error: %v)", backupErr, err)
		}

		return nil, fmt.Errorf("whitelist file was corrupted and backed up to %s: %w", backupPath, err)
	}

	// Validate whitelist state
	if err := ValidateWhitelistState(&state); err != nil {
		return nil, fmt.Errorf("invalid whitelist state: %w", err)
	}

	return &state, nil
}

// SaveWhitelistState saves a whitelist's state to its YAML file using atomic writes.
func SaveWhitelistState(ctx context.Context, state *WhitelistState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Validate whitelist state
	if err := ValidateWhitelistState(state); err != nil {
		return fmt.Errorf("invalid whitelist state: %w", err)
	}

	whitelistPath, err := GetWhitelistPath(state.Name)
	if err != nil {
		return fmt.Errorf("failed to get whitelist path: %w", err)
	}

	// Update timestamp
	state.UpdatedAt = time.Now()

	// Marshal to YAML
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal whitelist state: %w", err)
	}

	// Atomic write
	if err := AtomicWrite(whitelistPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write whitelist state: %w", err)
	}

	return nil
}

// DeleteWhitelistState deletes a whitelist's state file.
func DeleteWhitelistState(ctx context.Context, name string) error {
	if err := ValidateWhitelistName(name); err != nil {
		return err
	}

	whitelistPath, err := GetWhitelistPath(name)
	if err != nil {
		return fmt.Errorf("failed to get whitelist path: %w", err)
	}

	// Check if whitelist file exists
	if _, err := os.Stat(whitelistPath); os.IsNotExist(err) {
		return fmt.Errorf("whitelist %q does not exist", name)
	}

	// Delete file
	if err := os.Remove(whitelistPath); err != nil {
		return fmt.Errorf("failed to delete whitelist file: %w", err)
	}

	return nil
}

// WhitelistExists checks if a whitelist state file exists.
func WhitelistExists(ctx context.Context, name string) (bool, error) {
	if err := ValidateWhitelistName(name); err != nil {
		return false, err
	}

	whitelistPath, err := GetWhitelistPath(name)
	if err != nil {
		return false, fmt.Errorf("failed to get whitelist path: %w", err)
	}

	_, err = os.Stat(whitelistPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check whitelist file: %w", err)
	}

	return true, nil
}

// ListWhitelistStates returns a list of all whitelist state files.
func ListWhitelistStates(ctx context.Context) ([]string, error) {
	whitelistsDir, err := GetWhitelistsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get whitelists directory: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(whitelistsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read whitelists directory: %w", err)
	}

	// Extract whitelist names
	whitelists := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Remove .yaml extension
		if len(name) > 5 && name[len(name)-5:] == ".yaml" {
			whitelistName := name[:len(name)-5]
			whitelists = append(whitelists, whitelistName)
		}
	}

	return whitelists, nil
}

// ValidateWhitelistState validates a whitelist state.
func ValidateWhitelistState(state *WhitelistState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Validate name
	if err := ValidateWhitelistName(state.Name); err != nil {
		return fmt.Errorf("invalid whitelist name: %w", err)
	}

	// Validate players
	seenUUIDs := make(map[string]bool)
	for _, player := range state.Players {
		if err := ValidateUUID(player.UUID); err != nil {
			return fmt.Errorf("invalid player UUID: %w", err)
		}
		if err := ValidatePlayerName(player.Name); err != nil {
			return fmt.Errorf("invalid player name: %w", err)
		}

		// Check for duplicate UUIDs
		if seenUUIDs[player.UUID] {
			return fmt.Errorf("duplicate player UUID: %s", player.UUID)
		}
		seenUUIDs[player.UUID] = true
	}

	return nil
}

// AddPlayer adds a player to a whitelist.
func AddPlayer(ctx context.Context, whitelistName string, player PlayerInfo) error {
	// Validate player
	if err := ValidateUUID(player.UUID); err != nil {
		return fmt.Errorf("invalid player UUID: %w", err)
	}
	if err := ValidatePlayerName(player.Name); err != nil {
		return fmt.Errorf("invalid player name: %w", err)
	}

	// Load whitelist state (or create if doesn't exist)
	state, err := LoadWhitelistState(ctx, whitelistName)
	if err != nil {
		// If whitelist doesn't exist, create it
		if os.IsNotExist(err) || (err != nil && fmt.Sprintf("%v", err) == fmt.Sprintf("whitelist %q does not exist", whitelistName)) {
			state = NewWhitelistState(whitelistName)
		} else {
			return err
		}
	}

	// Check if player already exists
	for _, p := range state.Players {
		if p.UUID == player.UUID {
			return fmt.Errorf("player %q is already in whitelist", player.Name)
		}
	}

	// Set added timestamp if not set
	if player.AddedAt.IsZero() {
		player.AddedAt = time.Now()
	}

	state.Players = append(state.Players, player)

	return SaveWhitelistState(ctx, state)
}

// RemovePlayer removes a player from a whitelist by UUID.
func RemovePlayer(ctx context.Context, whitelistName, uuid string) error {
	if err := ValidateUUID(uuid); err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}

	state, err := LoadWhitelistState(ctx, whitelistName)
	if err != nil {
		return err
	}

	// Find and remove player
	found := false
	newPlayers := make([]PlayerInfo, 0, len(state.Players))
	for _, p := range state.Players {
		if p.UUID != uuid {
			newPlayers = append(newPlayers, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("player with UUID %q is not in whitelist", uuid)
	}

	state.Players = newPlayers

	return SaveWhitelistState(ctx, state)
}

// IsPlayerInWhitelist checks if a player is in a whitelist by UUID.
func IsPlayerInWhitelist(ctx context.Context, whitelistName, uuid string) (bool, error) {
	if err := ValidateUUID(uuid); err != nil {
		return false, fmt.Errorf("invalid UUID: %w", err)
	}

	state, err := LoadWhitelistState(ctx, whitelistName)
	if err != nil {
		return false, err
	}

	for _, p := range state.Players {
		if p.UUID == uuid {
			return true, nil
		}
	}

	return false, nil
}

// GetPlayer returns a player from a whitelist by UUID.
func GetPlayer(ctx context.Context, whitelistName, uuid string) (*PlayerInfo, error) {
	if err := ValidateUUID(uuid); err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	state, err := LoadWhitelistState(ctx, whitelistName)
	if err != nil {
		return nil, err
	}

	for _, p := range state.Players {
		if p.UUID == uuid {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("player with UUID %q is not in whitelist", uuid)
}

// ListPlayers returns all players in a whitelist.
func ListPlayers(ctx context.Context, whitelistName string) ([]PlayerInfo, error) {
	state, err := LoadWhitelistState(ctx, whitelistName)
	if err != nil {
		return nil, err
	}

	// Return a copy to prevent modification
	players := make([]PlayerInfo, len(state.Players))
	copy(players, state.Players)

	return players, nil
}
