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
- Comprehensive test suite with 83.7% coverage
- Viper configuration management with ~/.config/go-mc/config.yaml support

### Changed
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
