# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project initialization with README.md, CLAUDE.md, and development guidelines
- GitHub Issues and Milestones for all development phases
- GitHub Workflows for linting, testing, security scanning, and releases
- Makefile for local development
- golang-pro sub-agent for Go development tasks
- Cobra CLI framework with root command and version command (#1)
- Global flags: --config, --json, --quiet, --verbose with mutual exclusivity
- Structured logging with slog (log level adapts to flags)
- Command group structure: servers, users, whitelist, mods, system, config
- Comprehensive test suite with 80.9% coverage
- Viper configuration management with ~/.config/go-mc/config.yaml support
- YAML-based state management system (#2)
  - Config management with defaults and validation
  - Global state tracking (port allocation, server registry)
  - Per-server state persistence with full lifecycle tracking
  - Whitelist state management
  - Atomic file writes (temp file + rename pattern)
  - File locking with syscall.Flock() for concurrent safety
  - XDG Base Directory specification compliance
  - Automatic recovery from corrupted YAML files
  - Path traversal prevention and input validation
  - 80.5% test coverage for state package
- PID-based locking system to prevent concurrent execution (#3)
  - PID file at ~/.config/go-mc/go-mc.pid
  - Automatic stale PID cleanup for crashed processes
  - Signal handling (SIGINT/SIGTERM) for graceful shutdown
  - File-based locking with syscall.Flock()
  - Process existence validation
  - Clear error messages for concurrent execution attempts
- Podman/Docker client integration with auto-detection (#4)
  - Automatic runtime detection (Podman rootless → rootful → Docker)
  - Socket auto-discovery for Podman and Docker
  - User-friendly error messages with installation instructions
  - Context-aware timeouts for all operations
  - Connection health checks (Ping, Info)
  - Mock client for testing
  - Custom error types with runtime-specific guidance
  - 65.8% test coverage for container package
- Container lifecycle operations for Minecraft servers (#5)
  - CreateContainer with full configuration support (ports, volumes, resources, labels)
  - StartContainer and WaitForContainer for state management
  - StopContainer with graceful timeout (SIGTERM → SIGKILL)
  - RestartContainer operation
  - RemoveContainer with force and volume cleanup options
  - InspectContainer for detailed container information
  - ListContainers with filtering and pagination
  - Port mapping validation (1-65535 range)
  - Volume mount support with read-only/read-write options
  - Resource limits (memory parsing with docker/go-units, CPU quota)
  - Comprehensive error handling with custom error types
  - 44.8% unit test coverage + 100% integration test coverage
  - 2,300+ lines of implementation and tests

### Changed
- Go version upgraded from 1.21 to 1.22.6 (required by Podman v5 dependency)
- Replaced placeholder main.go with complete CLI implementation
- Fixed Makefile install-hooks target syntax error

### Fixed
- None

## [0.1.0] - 2025-01-20

### Added
- Initial project setup
- Complete project documentation (North Star README)
- Development guidelines (CLAUDE.md)
- CI/CD pipeline configuration
