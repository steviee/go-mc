package state

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the user configuration for go-mc.
type Config struct {
	Container ContainerConfig `yaml:"container"`
	Defaults  DefaultsConfig  `yaml:"defaults"`
	Ports     PortsConfig     `yaml:"ports"`
	Backups   BackupsConfig   `yaml:"backups"`
	TUI       TUIConfig       `yaml:"tui"`
	Mods      ModsConfig      `yaml:"mods"`
	Logging   LoggingConfig   `yaml:"logging"`
	Limits    LimitsConfig    `yaml:"limits"`
}

// ContainerConfig holds container runtime configuration.
type ContainerConfig struct {
	Runtime    string `yaml:"runtime"`
	Socket     string `yaml:"socket"`
	Image      string `yaml:"image"`
	PullPolicy string `yaml:"pull_policy"`
}

// DefaultsConfig holds default values for new servers.
type DefaultsConfig struct {
	Memory              string `yaml:"memory"`
	JavaVersion         int    `yaml:"java_version"`
	MinecraftVersion    string `yaml:"minecraft_version"`
	FabricLoaderVersion string `yaml:"fabric_loader_version"`
}

// PortsConfig holds port allocation configuration.
type PortsConfig struct {
	GamePortStart      int `yaml:"game_port_start"`
	RconPortStart      int `yaml:"rcon_port_start"`
	RconPasswordLength int `yaml:"rcon_password_length"`
}

// BackupsConfig holds backup configuration.
type BackupsConfig struct {
	Directory              string `yaml:"directory"`
	Compress               bool   `yaml:"compress"`
	KeepCount              int    `yaml:"keep_count"`
	AutoBackupBeforeUpdate bool   `yaml:"auto_backup_before_update"`
}

// TUIConfig holds TUI configuration.
type TUIConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	Theme           string        `yaml:"theme"`
}

// ModsConfig holds mod management configuration.
type ModsConfig struct {
	CacheDir                string `yaml:"cache_dir"`
	AutoResolveDependencies bool   `yaml:"auto_resolve_dependencies"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// LimitsConfig holds resource limits.
type LimitsConfig struct {
	MaxServers         int    `yaml:"max_servers"`
	MaxMemoryPerServer string `yaml:"max_memory_per_server"`
	MaxPorts           int    `yaml:"max_ports"`
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		Container: ContainerConfig{
			Runtime:    "podman",
			Socket:     "",
			Image:      "ghcr.io/itzg/minecraft-server:latest",
			PullPolicy: "missing",
		},
		Defaults: DefaultsConfig{
			Memory:              "2G",
			JavaVersion:         21,
			MinecraftVersion:    "latest",
			FabricLoaderVersion: "latest",
		},
		Ports: PortsConfig{
			GamePortStart:      25565,
			RconPortStart:      25575,
			RconPasswordLength: 16,
		},
		Backups: BackupsConfig{
			Directory:              "~/.config/go-mc/backups/archives/",
			Compress:               true,
			KeepCount:              5,
			AutoBackupBeforeUpdate: true,
		},
		TUI: TUIConfig{
			RefreshInterval: 1 * time.Second,
			Theme:           "default",
		},
		Mods: ModsConfig{
			CacheDir:                "~/.cache/go-mc/mods/",
			AutoResolveDependencies: true,
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "~/.config/go-mc/go-mc.log",
		},
		Limits: LimitsConfig{
			MaxServers:         50,
			MaxMemoryPerServer: "16G",
			MaxPorts:           100,
		},
	}
}

// LoadConfig loads the configuration from the config file.
// If the file doesn't exist, it creates a new one with defaults.
// If the file is corrupted, it backs up the corrupted file and creates a fresh one.
func LoadConfig(ctx context.Context) (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		cfg := DefaultConfig()
		if err := SaveConfig(ctx, cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Config file is corrupted, backup and create fresh
		backupPath := configPath + ".corrupted"
		if backupErr := os.Rename(configPath, backupPath); backupErr != nil {
			return nil, fmt.Errorf("config file is corrupted and failed to create backup: %w (original error: %v)", backupErr, err)
		}

		// Create fresh config
		cfg := DefaultConfig()
		if saveErr := SaveConfig(ctx, cfg); saveErr != nil {
			return nil, fmt.Errorf("config file was corrupted (backed up to %s), failed to save fresh config: %w (original error: %v)", backupPath, saveErr, err)
		}

		return cfg, nil
	}

	// Validate config
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to the config file using atomic writes.
func SaveConfig(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate config before saving
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write
	if err := AtomicWrite(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate container config
	if cfg.Container.Runtime != "podman" && cfg.Container.Runtime != "docker" {
		return fmt.Errorf("invalid container runtime: %q (must be podman or docker)", cfg.Container.Runtime)
	}

	if cfg.Container.Image == "" {
		return fmt.Errorf("container image cannot be empty")
	}

	// Validate defaults
	if err := ValidateMemory(cfg.Defaults.Memory); err != nil {
		return fmt.Errorf("invalid default memory: %w", err)
	}

	if err := ValidateJavaVersion(cfg.Defaults.JavaVersion); err != nil {
		return fmt.Errorf("invalid default Java version: %w", err)
	}

	// Validate ports
	if err := ValidatePort(cfg.Ports.GamePortStart); err != nil {
		return fmt.Errorf("invalid game port start: %w", err)
	}

	if err := ValidatePort(cfg.Ports.RconPortStart); err != nil {
		return fmt.Errorf("invalid rcon port start: %w", err)
	}

	if cfg.Ports.RconPasswordLength < 8 || cfg.Ports.RconPasswordLength > 32 {
		return fmt.Errorf("rcon password length must be between 8 and 32, got %d", cfg.Ports.RconPasswordLength)
	}

	// Validate backups
	if cfg.Backups.KeepCount < 0 {
		return fmt.Errorf("backup keep count must be >= 0, got %d", cfg.Backups.KeepCount)
	}

	// Validate TUI
	if cfg.TUI.RefreshInterval < 100*time.Millisecond {
		return fmt.Errorf("TUI refresh interval must be >= 100ms, got %v", cfg.TUI.RefreshInterval)
	}

	// Validate logging
	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLogLevels {
		if cfg.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %q (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	// Validate limits
	if cfg.Limits.MaxServers < 1 {
		return fmt.Errorf("max servers must be >= 1, got %d", cfg.Limits.MaxServers)
	}

	if err := ValidateMemory(cfg.Limits.MaxMemoryPerServer); err != nil {
		return fmt.Errorf("invalid max memory per server: %w", err)
	}

	if cfg.Limits.MaxPorts < 1 {
		return fmt.Errorf("max ports must be >= 1, got %d", cfg.Limits.MaxPorts)
	}

	return nil
}
