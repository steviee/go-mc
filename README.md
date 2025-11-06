# go-mc: Opinionated Minecraft Fabric Server Management CLI

**Repository:** https://github.com/steviee/go-mc
**Language:** Go 1.21+
**Target Platform:** Debian 12/13 (Linux x86_64/ARM64)
**Container Runtime:** Podman (rootless, default)
**License:** MIT

---

## 📋 Table of Contents

1. [Vision & Philosophy](#vision--philosophy)
2. [Quick Start](#quick-start)
3. [CLI Commands](#cli-commands)
4. [Architecture](#architecture)
5. [Project Structure](#project-structure)
6. [Roadmap](#roadmap)
7. [Development](#development)
8. [Contributing](#contributing)

---

## 🎯 Vision & Philosophy

### Vision

Create an **opinionated, production-ready CLI tool** for managing Minecraft Fabric servers on Debian systems using Podman containers. Built on the **Omakase principle** ("Chef's Choice") - sensible defaults with power-user customization available but not required.

### Core Principles

- **🍣 Omakase** - Works perfectly out-of-the-box, advanced options available
- **🔒 Rootless by Default** - Runs without root privileges using Podman
- **📝 YAML-Native** - Simple file-based state (~/.config/go-mc/)
- **🎯 Opinionated** - One right way: Fabric servers on Debian
- **🌐 Modrinth-First** - Integrated mod management with version matching
- **🔄 JSON API** - Machine-readable output for automation
- **🛠️ Zero Dependencies** - Single binary, self-installing dependencies

### Target Users

- **System Administrators** - Managing multiple Fabric servers on dedicated hardware
- **Self-hosters** - Running modded game servers for communities
- **Automation Engineers** - Infrastructure-as-Code for game server fleets

### Key Features

- ✅ Create Fabric servers with smart defaults (version, memory, ports)
- ✅ Start/stop/restart lifecycle management
- ✅ Real-time TUI dashboard (`servers top`)
- ✅ Modrinth mod search and auto-installation
- ✅ Global whitelist management with UUID resolution
- ✅ User/operator/ban management across servers
- ✅ Automatic version updates with dependency matching
- ✅ Backup and restore with rotation
- ✅ RCON command execution
- ✅ JSON output for all non-interactive commands
- ✅ Self-healing system setup and upgrades

---

## 🚀 Quick Start

### Installation

```bash
# Download and install (auto-detects architecture)
curl -fsSL https://raw.githubusercontent.com/steviee/go-mc/main/install.sh | sudo bash

# First-time setup (installs Podman, curl, git if needed)
go-mc system setup
```

### Create Your First Server

```bash
# Create with sensible defaults (latest Fabric, 2GB RAM, auto-port)
go-mc servers create survival

# Start the server
go-mc servers start survival

# Watch it in real-time
go-mc servers top
```

### Add Mods from Modrinth

```bash
# Search for mods (shows compatible versions)
go-mc mods search sodium --json

# Add mods to server (auto-resolves dependencies)
go-mc servers create modded --mods sodium,lithium,iris
```

### User Management

```bash
# Add user to global whitelist (auto-fetches UUID)
go-mc users add survival Steve

# Make someone an operator
go-mc users op survival Steve

# Enable whitelist on server
go-mc whitelist enable survival
```

---

## 🖥️ CLI Commands

### Command Structure

```bash
go-mc [global-flags] <group> <command> [flags] [arguments]
```

### Global Flags

```
--config, -c       Config file (default: ~/.config/go-mc/config.yaml)
--json             Output in JSON format (non-interactive commands)
--quiet, -q        Suppress non-error output
--verbose, -v      Verbose logging
--help, -h         Show help
--version          Show version info
```

---

## 📦 Command Groups

### `go-mc servers` - Server Management

Core server lifecycle operations.

#### `servers create <name>`

Create a new Fabric server instance with intelligent defaults.

**Flags:**
```
--version <version>          Minecraft version (default: latest stable)
--fabric-version <version>   Fabric loader version (default: latest compatible)
--memory <size>              Memory allocation (default: 2G)
--port <port>                Server port (default: auto-assign from 25565+)
--rcon-port <port>           RCON port (default: auto-assign from 25575+)
--rcon-password <password>   RCON password (default: auto-generated)
--world-seed <seed>          World seed (default: random)
--difficulty <level>         Difficulty: peaceful, easy, normal, hard (default: normal)
--gamemode <mode>            Gamemode: survival, creative, adventure (default: survival)
--max-players <count>        Max players (default: 20)
--whitelist <name>           Whitelist to use (default: "default")
--whitelist-enabled          Enable whitelist on creation (default: false)
--online-mode                Enable online mode (default: true)
--pvp                        Enable PVP (default: true)
--mods <slugs>               Comma-separated Modrinth mod slugs
--start                      Start server immediately after creation
--dry-run                    Show what would be created without doing it
```

**Examples:**
```bash
# Minimal - use all defaults
go-mc servers create survival

# Specific version with mods
go-mc servers create modded --version 1.20.4 --mods sodium,lithium,iris

# Performance server with high memory
go-mc servers create performance --memory 8G --max-players 50

# Create and start immediately
go-mc servers create test --start

# Preview without creating
go-mc servers create test --dry-run --json
```

**Output:**
```
Creating Minecraft Fabric server 'survival'...
  Minecraft:  1.20.4
  Fabric:     0.15.7
  Memory:     2G
  Port:       25565
  RCON:       25575 (password: xK9mP2vL8nQ4wR7z)

Pulling container image ghcr.io/itzg/minecraft-server:latest...
Creating Podman container go-mc-survival...
Writing configuration to ~/.config/go-mc/servers/survival.yaml

✓ Server 'survival' created successfully
  Container ID: a1b2c3d4e5f6

Next steps:
  Start:  go-mc servers start survival
  Logs:   go-mc servers logs survival -f
  Status: go-mc servers inspect survival
```

#### `servers list` (alias: `servers ls`, legacy: `ps`)

List all servers in tabular format.

**Flags:**
```
--all, -a          Show all servers (including stopped)
--filter, -f       Filter by status: running, stopped, created, error
--format           Output format: table, json, yaml
--no-header        Omit header row
--sort             Sort by: name, status, created, memory, cpu
```

**Output (table):**
```
NAME          STATUS      VERSION   PLAYERS   MEMORY      CPU    UPTIME    PORT
survival      running     1.20.4    3/20      1.8G/2G     12%    2d 5h     25565
creative      running     1.20.1    0/10      800M/2G     2%     3h 24m    25566
modded        stopped     1.19.4    -         -           -      -         25567
```

**Output (JSON):**
```json
{
  "servers": [
    {
      "name": "survival",
      "status": "running",
      "minecraft_version": "1.20.4",
      "fabric_version": "0.15.7",
      "players": {"online": 3, "max": 20},
      "resources": {
        "memory": {"used": "1.8G", "limit": "2G", "percent": 90},
        "cpu_percent": 12
      },
      "uptime": "2d5h",
      "port": 25565,
      "container_id": "a1b2c3d4e5f6"
    }
  ]
}
```

#### `servers top`

Interactive TUI dashboard with real-time monitoring (like `htop` or `lazydocker`).

**Features:**
- Real-time metrics (CPU, memory, players, TPS)
- Color-coded status indicators
- Sortable columns
- Quick actions via hotkeys
- Auto-refresh (configurable)
- Resource graphs

**Keyboard Shortcuts:**
```
↑/↓           Navigate servers
Enter         Show server details
s             Start selected server
x             Stop selected server
r             Restart selected server
l             View logs
d             Delete server (with confirmation)
u             Update server
q/Ctrl+C      Quit
?             Show help
```

#### `servers start <name...>`

Start one or more stopped servers.

**Flags:**
```
--all, -a          Start all stopped servers
--wait, -w         Wait until server is fully started
--timeout <dur>    Timeout for wait (default: 5m)
```

**Examples:**
```bash
go-mc servers start survival
go-mc servers start server1 server2 server3
go-mc servers start --all
go-mc servers start survival --wait --json
```

#### `servers stop <name...>`

Stop one or more running servers gracefully.

**Flags:**
```
--all, -a          Stop all running servers
--force, -f        Force stop (skip graceful shutdown)
--timeout <dur>    Graceful shutdown timeout (default: 30s)
```

#### `servers restart <name...>`

Restart one or more servers.

**Flags:**
```
--all, -a          Restart all servers
--wait, -w         Wait until restart is complete
```

#### `servers rm <name...>` (alias: `servers remove`, `servers delete`)

Remove one or more servers (with confirmation).

**Flags:**
```
--force, -f        Skip confirmation prompt
--volumes, -v      Remove associated volumes (world data)
--keep-backups     Keep backups even if removing volumes
```

**Examples:**
```bash
go-mc servers rm old-server
go-mc servers rm test --force --volumes
```

#### `servers logs <name>`

View server logs with filtering and streaming.

**Flags:**
```
--follow, -f       Follow log output (stream)
--tail <n>         Show last N lines (default: 100)
--since <time>     Show logs since timestamp/duration
--timestamps, -t   Show timestamps
--grep <pattern>   Filter logs by regex pattern
```

**Examples:**
```bash
go-mc servers logs survival
go-mc servers logs survival -f
go-mc servers logs survival --tail 50 --grep "ERROR"
go-mc servers logs survival --since 1h --timestamps
```

#### `servers exec <name> <command>`

Execute RCON command on running server.

**Examples:**
```bash
go-mc servers exec survival "say Hello players!"
go-mc servers exec survival "whitelist add Steve"
go-mc servers exec survival "gamemode creative Alex"
go-mc servers exec survival "save-all"
go-mc servers exec survival "list" --json
```

#### `servers inspect <name>`

Show detailed information about a server.

**Flags:**
```
--format           Output format: yaml, json (default: yaml)
```

**Output (YAML):**
```yaml
name: survival
id: 550e8400-e29b-41d4-a716-446655440000
status: running
container_id: a1b2c3d4e5f6789

minecraft:
  version: 1.20.4
  fabric_version: 0.15.7

resources:
  memory:
    limit: 2G
    used: 1.8G
    percent: 90
  cpu_percent: 12

network:
  server_port: 25565
  rcon_port: 25575

gameplay:
  difficulty: normal
  gamemode: survival
  max_players: 20
  whitelist_enabled: true
  online_mode: true
  pvp_enabled: true
  world_seed: "12345"

players:
  online: 3
  max: 20
  list:
    - username: Steve
      uuid: 069a79f4-44e9-4726-a5be-fca90e38aaf5
      connected_at: 2025-01-18T14:20:00Z
    - username: Alex
      uuid: ec561538-f3fd-461d-aff5-086b22154bce
      connected_at: 2025-01-18T15:10:00Z

mods:
  count: 15
  list:
    - name: Sodium
      slug: sodium
      version: 0.5.5
      modrinth_id: AANobbMI
      file: sodium-fabric-mc1.20.4-0.5.5.jar
    - name: Lithium
      slug: lithium
      version: 0.12.0
      modrinth_id: gvQqBUqZ
      file: lithium-fabric-mc1.20.4-0.12.0.jar

timestamps:
  created_at: 2025-01-15T10:30:45Z
  updated_at: 2025-01-18T14:22:10Z
  started_at: 2025-01-18T14:20:00Z
  uptime: 2d5h23m

backups:
  count: 5
  latest: 2025-01-18T03:00:00Z
```

#### `servers update <name>`

Update Minecraft/Fabric version or mods.

**Flags:**
```
--version <version>    Update to specific Minecraft version
--latest               Update to latest Minecraft + Fabric
--mods-only            Update mods only (preserve MC version)
--backup               Create backup before update (default: true)
--restart              Restart after update (default: false)
```

**Examples:**
```bash
# Update Minecraft version (auto-updates Fabric + mods)
go-mc servers update survival --version 1.20.5

# Update to latest everything
go-mc servers update survival --latest --restart

# Update only mods to compatible versions
go-mc servers update modded --mods-only
```

#### `servers backup <name>`

Create backup of server data.

**Flags:**
```
--all, -a          Backup all servers
--output, -o       Output directory (default: ~/.config/go-mc/backups/)
--compress         Compress backup (default: true)
--keep <n>         Keep last N backups (default: 5)
```

**Examples:**
```bash
go-mc servers backup survival
go-mc servers backup --all
go-mc servers backup survival --output /mnt/backups/
```

#### `servers restore <name> <backup-id>`

Restore server from backup.

**Flags:**
```
--force, -f        Overwrite existing data without confirmation
--stop             Stop server before restore (default: true)
--start            Start server after restore (default: false)
```

**Examples:**
```bash
# List available backups
go-mc servers backup survival --list

# Restore from specific backup
go-mc servers restore survival backup-2025-01-18-03-00-00
```

---

### `go-mc users` - User Management

Manage players across servers with UUID resolution.

#### `users add <server> <username>`

Add user to server's whitelist (auto-fetches UUID from Mojang/Microsoft API).

**Flags:**
```
--uuid <uuid>      Manually specify UUID (skip API lookup)
--global           Add to global "default" whitelist
--whitelist <name> Add to named whitelist
```

**Examples:**
```bash
# Add to specific server
go-mc users add survival Steve

# Add to global whitelist (syncs to all servers using "default")
go-mc users add --global Steve

# Manually specify UUID
go-mc users add survival Steve --uuid 069a79f4-44e9-4726-a5be-fca90e38aaf5
```

#### `users remove <server> <username>`

Remove user from server's whitelist.

**Flags:**
```
--global           Remove from global whitelist
--whitelist <name> Remove from named whitelist
```

#### `users list <server>`

List all users on server's whitelist.

**Flags:**
```
--global           List global whitelist users
--whitelist <name> List named whitelist users
--format           Output format: table, json
```

**Output:**
```
USERNAME   UUID                                   ADDED
Steve      069a79f4-44e9-4726-a5be-fca90e38aaf5   2025-01-15 10:30:45
Alex       ec561538-f3fd-461d-aff5-086b22154bce   2025-01-16 14:20:30
```

#### `users ban <server> <username>`

Ban user from server.

**Flags:**
```
--reason <text>    Ban reason
--expires <time>   Temporary ban duration (e.g., 24h, 7d)
```

**Examples:**
```bash
go-mc users ban survival Griefer --reason "Destroying builds"
go-mc users ban survival Spammer --reason "Spam" --expires 7d
```

#### `users unban <server> <username>`

Remove user from ban list.

#### `users op <server> <username>`

Make user a server operator.

**Flags:**
```
--level <1-4>      Operator permission level (default: 4)
```

#### `users deop <server> <username>`

Remove operator privileges from user.

#### `users kick <server> <username>`

Kick user from running server (via RCON).

**Flags:**
```
--reason <text>    Kick reason
```

---

### `go-mc whitelist` - Whitelist Management

Manage global and named whitelists that sync across servers.

#### `whitelist create <name>`

Create a named whitelist.

**Examples:**
```bash
go-mc whitelist create vip-players
go-mc whitelist create staff
```

#### `whitelist list`

List all whitelists.

**Output:**
```
NAME          USERS   SERVERS
default       12      3
vip-players   5       1
staff         8       3
```

#### `whitelist add <name> <username>`

Add user to named whitelist (auto-syncs to servers using this whitelist).

**Examples:**
```bash
go-mc whitelist add default Steve
go-mc whitelist add vip-players Alex
```

#### `whitelist remove <name> <username>`

Remove user from named whitelist.

#### `whitelist enable <server>`

Enable whitelist enforcement on server.

**Flags:**
```
--whitelist <name> Specify which whitelist to use (default: "default")
```

**Examples:**
```bash
go-mc whitelist enable survival
go-mc whitelist enable vip-server --whitelist vip-players
```

#### `whitelist disable <server>`

Disable whitelist enforcement on server (server becomes public).

#### `whitelist sync <name>`

Manually sync whitelist to all servers using it.

**Examples:**
```bash
# Sync default whitelist to all servers
go-mc whitelist sync default

# Sync specific whitelist
go-mc whitelist sync vip-players
```

---

### `go-mc mods` - Modrinth Mod Management

Search, install, and manage Fabric mods from Modrinth.

#### `mods search <query>`

Search Modrinth for Fabric mods.

**Flags:**
```
--version, -v <version>    Filter by Minecraft version
--limit, -l <n>            Max results (default: 20, max: 100)
--sort <field>             Sort by: relevance, downloads, updated (default: relevance)
```

**Examples:**
```bash
# Search for performance mods
go-mc mods search sodium

# Search for 1.21.1 compatible mods
go-mc mods search shaders --version 1.21.1 --limit 10

# Sort by downloads
go-mc mods search optimization --sort downloads --limit 50

# JSON output for scripting (using global --json flag)
go-mc mods search fabric-api --json
```

**Output (table):**
```
SLUG                 NAME                      DOWNLOADS  DESCRIPTION
----------------------------------------------------------------------------------------------------
sodium               Sodium                    86.1M      The fastest and most compatible rende...
lithium              Lithium                   46.9M      No-compromises game logic/server opti...
iris                 Iris Shaders              24.3M      A modern shaders mod for Minecraft in...

Found 3 result(s).
```

**Output (JSON):**
```json
{
  "status": "success",
  "data": {
    "results": [
      {
        "slug": "sodium",
        "title": "Sodium",
        "description": "The fastest and most compatible rendering optimization mod for Minecraft",
        "project_id": "AANobbMI",
        "project_type": "mod",
        "downloads": 86123456,
        "icon_url": "https://cdn.modrinth.com/data/AANobbMI/icon.png",
        "author": "JellySquid",
        "categories": ["optimization", "fabric"]
      }
    ],
    "count": 1,
    "total": 156,
    "limit": 20,
    "offset": 0
  }
}
```

#### `mods install <server> <slug...>`

Install mods from Modrinth to server (auto-resolves dependencies).

**Flags:**
```
--version <version>    Specific mod version (default: latest compatible)
--skip-deps            Don't install dependencies
--restart              Restart server after installation
```

**Examples:**
```bash
# Install single mod
go-mc mods install survival sodium

# Install multiple mods (auto-resolves fabric-api dependency)
go-mc mods install modded sodium lithium iris

# Specific version
go-mc mods install survival sodium --version 0.5.5
```

#### `mods list <server>`

List installed mods on server.

**Flags:**
```
--format               Output format: table, json
--check-updates        Check for available updates
```

**Output:**
```
NAME          VERSION   MC VERSION   UPDATES
Fabric API    0.92.0    1.20.4       0.92.1 available
Sodium        0.5.5     1.20.4       Up to date
Lithium       0.12.0    1.20.4       0.12.1 available
```

#### `mods update <server> <slug>`

Update specific mod to latest compatible version.

**Flags:**
```
--all, -a          Update all mods on server
--version <ver>    Update to specific version
--restart          Restart server after update
```

**Examples:**
```bash
# Update single mod
go-mc mods update survival sodium

# Update all mods
go-mc mods update survival --all --restart
```

#### `mods remove <server> <slug...>` (alias: `mods rm`)

Remove mods from server.

**Flags:**
```
--restart          Restart server after removal
```

---

### `go-mc system` - System Management

Manage go-mc installation and dependencies.

#### `system setup`

First-time setup - installs Podman, curl, git if needed (requires sudo).

**Flags:**
```
--non-interactive  Skip all prompts, assume yes to all (for scripts)
--skip-deps        Skip dependency installation (if already installed)
--force            Force setup even if already configured
```

**Examples:**
```bash
# Initial setup (interactive with prompts)
go-mc system setup

# Automated setup (for scripts/CI)
go-mc system setup --non-interactive

# Re-run setup without reinstalling dependencies
go-mc system setup --skip-deps --force
```

**Operations:**
1. Check OS compatibility (Debian 12/13 only)
2. Detect missing dependencies (Podman, curl, git)
3. Install missing dependencies via apt-get (requires sudo)
4. Configure Podman for rootless operation (subuid/subgid, systemd socket)
5. Create XDG directory structure (~/.config/go-mc/, ~/.local/share/go-mc/)
6. Generate default config.yaml with sensible defaults
7. Initialize global state with empty server registry
8. Pull default container image (itzg/minecraft-server:latest)

#### `system upgrade`

Upgrade system dependencies (Podman, container images).

**Flags:**
```
--deps-only        Only upgrade system packages
--images-only      Only update container images
--dry-run          Show what would be upgraded
```

**Examples:**
```bash
# Upgrade everything
sudo go-mc system upgrade

# Only update container images (no sudo needed)
go-mc system upgrade --images-only
```

#### `system status`

Show system health and configuration.

**Output:**
```
go-mc System Status
═══════════════════════════════════════════════

Version:         v1.0.0 (commit: a1b2c3d)
Go Version:      1.21.5
OS:              Debian 12 (bookworm)
Architecture:    linux/amd64

Container Runtime:
  Engine:        Podman 4.3.1
  Rootless:      ✓ Enabled
  Socket:        /run/user/1000/podman/podman.sock
  Images:        3 (2.1 GB)
  Containers:    4 running, 2 stopped

Configuration:
  Config File:   ~/.config/go-mc/config.yaml
  State Dir:     ~/.config/go-mc/
  Backups Dir:   ~/.config/go-mc/backups/

Servers:
  Total:         6
  Running:       4
  Stopped:       2

Dependencies:
  Podman:        ✓ 4.3.1
  curl:          ✓ 7.88.1
  git:           ✓ 2.39.2

Health:          ✓ All systems operational
```

#### `system cleanup`

Clean up unused container images, volumes, and old backups.

**Flags:**
```
--all, -a          Clean all (images, volumes, backups)
--images           Clean unused images
--volumes          Clean unused volumes
--backups          Clean old backups (>180 days)
--force, -f        Skip confirmation
--dry-run          Show what would be cleaned
```

**Examples:**
```bash
# Preview cleanup
go-mc system cleanup --dry-run

# Clean everything
go-mc system cleanup --all --force

# Only clean old backups
go-mc system cleanup --backups
```

#### `system doctor`

Diagnose and fix common issues.

**Checks:**
- Podman installation and configuration
- Rootless Podman setup
- Port availability
- File permissions
- Config file validity
- Container image status
- Network connectivity
- Disk space

**Examples:**
```bash
go-mc system doctor
go-mc system doctor --json
```

---

### `go-mc config` - Configuration Management

Manage global configuration.

#### `config get <key>`

Get configuration value.

**Examples:**
```bash
go-mc config get defaults.memory
go-mc config get container.runtime
```

#### `config set <key> <value>`

Set configuration value.

**Examples:**
```bash
go-mc config set defaults.memory 4G
go-mc config set container.runtime docker
go-mc config set backups.keep_count 10
```

#### `config list`

List all configuration values.

**Flags:**
```
--format           Output format: yaml, json
--defaults         Show default values
```

#### `config edit`

Open config file in $EDITOR.

#### `config reset`

Reset configuration to defaults (with confirmation).

**Flags:**
```
--force, -f        Skip confirmation
```

---

### `go-mc version`

Show version information.

**Output:**
```
go-mc version v1.0.0
  Build:       2025-01-20 14:32:15
  Commit:      a1b2c3d
  Go:          1.21.5
  OS/Arch:     linux/amd64
  Podman:      4.3.1 (API 4.0)
```

---

## 🏗️ Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────┐
│                     go-mc CLI                           │
│                  (Single Binary)                        │
├─────────────────────────────────────────────────────────┤
│  CLI Layer     (cobra + command routing)                │
│  TUI Layer     (bubbletea + lipgloss)                   │
│  Service Layer (business logic)                         │
│  State Layer   (YAML file management)                   │
│  Podman Client (Podman API via Go SDK)                  │
│  Modrinth API  (mod search + version resolution)        │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│               Podman (Rootless)                         │
├─────────────────────────────────────────────────────────┤
│  Container: go-mc-survival                              │
│  Container: go-mc-creative                              │
│  Container: go-mc-modded                                │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│           YAML State & Data                             │
├─────────────────────────────────────────────────────────┤
│  ~/.config/go-mc/                                       │
│    ├── config.yaml        (global config)               │
│    ├── state.yaml         (global state)                │
│    ├── whitelists/                                      │
│    │   ├── default.yaml                                 │
│    │   └── vip-players.yaml                             │
│    ├── servers/                                         │
│    │   ├── survival.yaml                                │
│    │   ├── creative.yaml                                │
│    │   └── modded.yaml                                  │
│    └── backups/                                         │
│        ├── registry.yaml                                │
│        └── archives/                                    │
│                                                          │
│  /var/lib/go-mc/         (server data - optional)       │
│    └── servers/                                         │
│        ├── survival/                                    │
│        │   ├── data/     (world, config, logs)         │
│        │   └── mods/     (mod .jar files)               │
│        └── creative/                                    │
└─────────────────────────────────────────────────────────┘
```

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Language** | Go 1.21+ | Core application |
| **CLI Framework** | [cobra](https://github.com/spf13/cobra) | Command structure |
| **TUI Framework** | [bubbletea](https://github.com/charmbracelet/bubbletea) | Interactive dashboard |
| **TUI Styling** | [lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| **Container Runtime** | [Podman](https://podman.io) | Rootless containers |
| **Container Client** | [containers/podman](https://github.com/containers/podman) | Podman Go API |
| **Config/State** | [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | YAML parsing |
| **Logging** | [slog](https://pkg.go.dev/log/slog) | Structured logging |
| **RCON** | [gorcon/rcon](https://github.com/gorcon/rcon-cli) | Minecraft RCON |
| **HTTP Client** | [net/http](https://pkg.go.dev/net/http) | Modrinth API |
| **Testing** | [testify](https://github.com/stretchr/testify) | Test framework |

### State Management (YAML)

#### File Structure

```
~/.config/go-mc/
├── config.yaml              # User configuration
├── state.yaml               # Global state (ports, locks)
├── whitelists/
│   ├── default.yaml         # Default whitelist (synced to all servers)
│   └── <name>.yaml          # Named whitelists
├── servers/
│   └── <server-name>.yaml   # Per-server config + state
└── backups/
    ├── registry.yaml        # Backup metadata
    └── archives/            # Compressed backups (.tar.gz)
```

#### config.yaml

```yaml
# go-mc configuration (~/.config/go-mc/config.yaml)

# Container runtime
container:
  runtime: podman          # podman or docker
  socket: ""               # Auto-detect if empty
  image: ghcr.io/itzg/minecraft-server:latest
  network: go-mc-network
  auto_pull: true

# Default server settings (Omakase defaults)
defaults:
  memory: 2G
  minecraft_version: latest
  fabric_version: latest   # Latest compatible with MC version
  difficulty: normal
  gamemode: survival
  max_players: 20
  online_mode: true
  pvp: true
  whitelist_enabled: false
  whitelist_name: default
  port_start: 25565
  rcon_port_start: 25575
  rcon_password_length: 16

# Backup settings
backups:
  directory: ~/.config/go-mc/backups/archives/
  compress: true
  keep_count: 5
  auto_backup_before_update: true

# TUI settings
tui:
  refresh_interval: 1s
  theme: default
  colors:
    running: green
    stopped: gray
    error: red

# Modrinth API
modrinth:
  base_url: https://api.modrinth.com/v2
  timeout: 30s
  auto_resolve_dependencies: true

# Logging
logging:
  level: info
  file: ~/.config/go-mc/go-mc.log

# Resource limits
limits:
  max_servers: 50
  max_memory_per_server: 16G
  disk_quota: 100G

# Cleanup settings
cleanup:
  unused_images_after: 30d
  stopped_servers_after: 90d
  old_backups_after: 180d
```

#### state.yaml

```yaml
# Global state (~/.config/go-mc/state.yaml)

allocated_ports:
  - 25565
  - 25566
  - 25575
  - 25576

server_registry:
  - name: survival
    file: servers/survival.yaml
  - name: creative
    file: servers/creative.yaml

lock:
  pid: 12345
  timestamp: 2025-01-18T14:20:00Z

last_cleanup: 2025-01-17T03:00:00Z
last_update_check: 2025-01-18T10:00:00Z
```

#### servers/<name>.yaml

```yaml
# Server configuration (~/.config/go-mc/servers/survival.yaml)

name: survival
id: 550e8400-e29b-41d4-a716-446655440000
status: running
container_id: a1b2c3d4e5f6789

minecraft:
  version: 1.20.4
  fabric_version: 0.15.7

resources:
  memory: 2G

network:
  server_port: 25565
  rcon_port: 25575
  rcon_password: xK9mP2vL8nQ4wR7z  # Could be encrypted

gameplay:
  difficulty: normal
  gamemode: survival
  max_players: 20
  whitelist_enabled: true
  whitelist_name: default
  online_mode: true
  pvp_enabled: true
  world_seed: "12345"

paths:
  data: /var/lib/go-mc/servers/survival/data
  mods: /var/lib/go-mc/servers/survival/mods

mods:
  - name: Fabric API
    slug: fabric-api
    version: 0.92.0+1.20.4
    modrinth_id: P7dR8mSH
    file: fabric-api-0.92.0+1.20.4.jar
  - name: Sodium
    slug: sodium
    version: 0.5.5
    modrinth_id: AANobbMI
    file: sodium-fabric-mc1.20.4-0.5.5.jar

timestamps:
  created_at: 2025-01-15T10:30:45Z
  updated_at: 2025-01-18T14:22:10Z
  started_at: 2025-01-18T14:20:00Z

operators:
  - username: Steve
    uuid: 069a79f4-44e9-4726-a5be-fca90e38aaf5
    level: 4

banned_players: []
```

#### whitelists/default.yaml

```yaml
# Global whitelist (~/.config/go-mc/whitelists/default.yaml)

name: default
created_at: 2025-01-15T10:00:00Z
updated_at: 2025-01-18T14:30:00Z

users:
  - username: Steve
    uuid: 069a79f4-44e9-4726-a5be-fca90e38aaf5
    added_at: 2025-01-15T10:30:45Z
  - username: Alex
    uuid: ec561538-f3fd-461d-aff5-086b22154bce
    added_at: 2025-01-16T14:20:30Z

# Servers using this whitelist
servers:
  - survival
  - creative
```

#### backups/registry.yaml

```yaml
# Backup registry (~/.config/go-mc/backups/registry.yaml)

backups:
  - id: backup-2025-01-18-03-00-00
    server: survival
    filename: survival-2025-01-18-03-00-00.tar.gz
    file_path: ~/.config/go-mc/backups/archives/survival-2025-01-18-03-00-00.tar.gz
    size_bytes: 524288000
    compressed: true
    created_at: 2025-01-18T03:00:00Z
  - id: backup-2025-01-17-03-00-00
    server: survival
    filename: survival-2025-01-17-03-00-00.tar.gz
    file_path: ~/.config/go-mc/backups/archives/survival-2025-01-17-03-00-00.tar.gz
    size_bytes: 520192000
    compressed: true
    created_at: 2025-01-17T03:00:00Z
```

### Concurrency & Locking

**Single Instance Enforcement:**
- PID file at `~/.config/go-mc/go-mc.pid`
- Prevents multiple concurrent `go-mc` processes
- Uses `syscall.Flock()` for atomic file locking
- Stale PID cleanup on startup (check if process exists)

**YAML File Operations:**
- Atomic writes: write to temp file → rename
- Read-modify-write pattern with file locks
- Retry logic for transient failures

---

## 📁 Project Structure

```
go-mc/
├── .claude/
│   ├── agents/
│   │   └── golang-pro.md         # Go specialist sub-agent
│   └── CLAUDE.md                 # Development rules for Claude Code
│
├── cmd/
│   └── go-mc/
│       └── main.go               # Entry point
│
├── internal/
│   ├── cli/                      # CLI commands
│   │   ├── root.go
│   │   ├── servers/              # Server management commands
│   │   │   ├── create.go
│   │   │   ├── list.go
│   │   │   ├── start.go
│   │   │   ├── stop.go
│   │   │   ├── restart.go
│   │   │   ├── rm.go
│   │   │   ├── top.go
│   │   │   ├── logs.go
│   │   │   ├── exec.go
│   │   │   ├── inspect.go
│   │   │   ├── update.go
│   │   │   ├── backup.go
│   │   │   └── restore.go
│   │   ├── users/                # User management commands
│   │   │   ├── add.go
│   │   │   ├── remove.go
│   │   │   ├── list.go
│   │   │   ├── ban.go
│   │   │   ├── unban.go
│   │   │   ├── op.go
│   │   │   ├── deop.go
│   │   │   └── kick.go
│   │   ├── whitelist/            # Whitelist commands
│   │   │   ├── create.go
│   │   │   ├── list.go
│   │   │   ├── add.go
│   │   │   ├── remove.go
│   │   │   ├── enable.go
│   │   │   ├── disable.go
│   │   │   └── sync.go
│   │   ├── mods/                 # Modrinth mod commands
│   │   │   ├── search.go
│   │   │   ├── install.go
│   │   │   ├── list.go
│   │   │   ├── update.go
│   │   │   └── remove.go
│   │   ├── system/               # System management
│   │   │   ├── setup.go
│   │   │   ├── upgrade.go
│   │   │   ├── status.go
│   │   │   ├── cleanup.go
│   │   │   └── doctor.go
│   │   ├── config/               # Config commands
│   │   │   ├── get.go
│   │   │   ├── set.go
│   │   │   ├── list.go
│   │   │   ├── edit.go
│   │   │   └── reset.go
│   │   └── version.go
│   │
│   ├── tui/                      # Terminal UI
│   │   ├── dashboard.go
│   │   ├── models.go
│   │   ├── views.go
│   │   ├── styles.go
│   │   ├── table.go
│   │   └── events.go
│   │
│   ├── service/                  # Business logic
│   │   ├── server.go
│   │   ├── lifecycle.go
│   │   ├── container.go
│   │   ├── rcon.go
│   │   ├── backup.go
│   │   ├── update.go
│   │   ├── metrics.go
│   │   ├── cleanup.go
│   │   ├── user.go
│   │   ├── whitelist.go
│   │   └── modrinth.go
│   │
│   ├── state/                    # YAML state management
│   │   ├── config.go
│   │   ├── state.go
│   │   ├── server.go
│   │   ├── whitelist.go
│   │   ├── backup.go
│   │   └── lock.go
│   │
│   ├── container/                # Podman integration
│   │   ├── client.go
│   │   ├── container.go
│   │   ├── image.go
│   │   ├── network.go
│   │   ├── volume.go
│   │   └── stats.go
│   │
│   ├── modrinth/                 # Modrinth API client
│   │   ├── client.go
│   │   ├── search.go
│   │   ├── version.go
│   │   └── dependencies.go
│   │
│   ├── minecraft/                # Minecraft-specific
│   │   ├── version.go
│   │   ├── rcon.go
│   │   ├── properties.go
│   │   ├── whitelist.go
│   │   ├── ops.go
│   │   └── uuid.go
│   │
│   ├── util/                     # Utilities
│   │   ├── logger.go
│   │   ├── format.go
│   │   ├── ports.go
│   │   ├── fs.go
│   │   ├── archive.go
│   │   ├── password.go
│   │   └── json.go
│   │
│   └── models/                   # Data models
│       ├── server.go
│       ├── backup.go
│       ├── whitelist.go
│       ├── user.go
│       └── mod.go
│
├── test/                         # Integration tests
│   ├── server_test.go
│   ├── lifecycle_test.go
│   ├── backup_test.go
│   └── modrinth_test.go
│
├── scripts/                      # Build scripts
│   ├── build.sh
│   ├── install.sh
│   └── uninstall.sh
│
├── .github/
│   └── workflows/
│       ├── lint.yml
│       ├── test.yml
│       ├── security.yml
│       └── release.yml
│
├── go.mod
├── go.sum
├── Makefile
├── README.md                     # This file
├── LICENSE
└── .gitignore
```

---

## 🗓️ Roadmap

### Phase 0: Project Setup ✅
- [x] Initialize repository
- [x] Set up Go module
- [x] Create README.md (North Star document)
- [x] Create CLAUDE.md (Development guidelines)
- [x] Set up GitHub Issues + Milestones
- [x] Configure GitHub workflows (lint, test, security, release)

### Phase 1: Core CLI & State Management
- [x] Set up Cobra CLI framework
- [ ] Implement YAML state management
- [x] Create config loading/validation
- [ ] Implement PID-based locking
- [x] Add structured logging
- [x] `version` and `config` commands (placeholders)

### Phase 2: Podman Integration
- [ ] Initialize Podman client
- [ ] Container lifecycle operations (create/start/stop/remove)
- [ ] Network management
- [ ] Volume management
- [ ] Stats streaming
- [ ] Port allocation strategy

### Phase 3: Server Lifecycle Commands
- [x] `servers create` command
- [x] `servers start/stop/restart` commands
- [ ] `servers rm` command
- [x] `servers list` (ps) command
- [x] Server state persistence in YAML
- [x] Legacy command aliases

### Phase 4: Logs & Inspect
- [ ] `servers logs` with streaming and filtering
- [ ] `servers inspect` with detailed info
- [ ] Log parsing and formatting

### Phase 5: RCON Integration
- [ ] RCON client implementation
- [ ] `servers exec` command
- [ ] Player count/TPS retrieval
- [ ] Graceful shutdown via RCON

### Phase 6: Modrinth Integration
- [x] Modrinth API client
- [x] `mods search` command
- [x] Version matching (MC + Fabric)
- [ ] `mods install` with dependency resolution
- [ ] `mods list/update/remove` commands

### Phase 7: User & Whitelist Management
- [ ] UUID lookup via Mojang/Microsoft API
- [ ] Global whitelist management
- [ ] `users add/remove/list` commands
- [ ] `users ban/unban/op/deop` commands
- [ ] `whitelist` commands
- [ ] Whitelist synchronization

### Phase 8: TUI Dashboard
- [ ] Bubbletea setup
- [ ] Real-time server list
- [ ] Interactive navigation
- [ ] Resource graphs
- [ ] Event log panel
- [ ] Quick actions (start/stop/logs)

### Phase 9: Update & Backup
- [ ] Version update logic
- [ ] `servers update` command
- [ ] Backup creation (tar.gz)
- [ ] `servers backup/restore` commands
- [ ] Backup rotation

### Phase 10: System Management
- [x] `system setup` (first-time installation)
- [x] Dependency installation (Podman, curl, git)
- [ ] `system upgrade` command
- [ ] `system status` command
- [ ] `system cleanup` command
- [ ] `system doctor` diagnostics

### Phase 11: Polish & Testing
- [ ] Comprehensive unit tests (70%+ coverage)
- [ ] Integration tests (happy path)
- [ ] Error handling improvements
- [ ] Performance optimization
- [ ] Documentation updates

### Phase 12: Release v1.0
- [ ] Build for amd64/arm64
- [ ] GitHub Release with binaries
- [ ] Installation script
- [ ] Debian package (.deb) - optional

### Future Enhancements (v1.1+)
- [ ] Multi-node support (remote server management)
- [ ] Scheduled backups (cron-like)
- [ ] Auto-update mechanism
- [ ] Prometheus metrics exporter
- [ ] Discord webhook notifications
- [ ] Plugin system

---

## 🛠️ Development

### Prerequisites

- **Go 1.21+**
- **Podman 4.0+** (or Docker with config)
- **Debian 12/13** (for production use)
- **make** (optional, for build automation)

### Building from Source

```bash
# Clone repository
git clone https://github.com/steviee/go-mc.git
cd go-mc

# Build
make build

# Or directly with go
go build -o go-mc ./cmd/go-mc

# Install locally
sudo make install
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Podman)
make test-integration

# Test coverage
make coverage

# Lint
make lint
```

### Development Workflow

1. **Create feature branch**: `git checkout -b feature/issue-123-description`
2. **Make changes**: Follow [CLAUDE.md](.claude/CLAUDE.md) guidelines
3. **Run tests**: `make test lint`
4. **Commit**: Use conventional commits (`feat:`, `fix:`, etc.)
5. **Push**: `git push origin feature/issue-123-description`
6. **Create PR**: Reference issue number, ensure CI passes
7. **Update docs**: Update README.md if needed
8. **Merge**: Squash and merge to main

### Using golang-pro Sub-Agent

All Go development tasks use the `golang-pro` sub-agent for idiomatic, high-performance code.

**Key Standards:**
- `gofmt` formatting (enforced by pre-commit)
- `golangci-lint` passes
- 70%+ test coverage
- Context propagation
- Comprehensive error handling
- Table-driven tests

See [.claude/CLAUDE.md](.claude/CLAUDE.md) for full guidelines.

---

## 🤝 Contributing

Contributions are welcome! This project uses **GitHub Issues** as the single source of truth for all tasks and features.

### How to Contribute

1. **Browse Issues**: Check [GitHub Issues](https://github.com/steviee/go-mc/issues) for open tasks
2. **Comment**: Comment on an issue to claim it
3. **Fork & Branch**: Create a feature branch
4. **Follow Guidelines**: Read [.claude/CLAUDE.md](.claude/CLAUDE.md)
5. **Submit PR**: Reference the issue, ensure tests pass
6. **Review**: Wait for review and address feedback

### Reporting Bugs

- Use the bug report template
- Include OS, Go version, Podman version
- Provide steps to reproduce
- Attach logs if applicable

### Feature Requests

- Check if issue already exists
- Describe use case clearly
- Explain how it aligns with project philosophy

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

---

## 🙏 Acknowledgments

- [itzg/docker-minecraft-server](https://github.com/itzg/docker-minecraft-server) - Excellent container image
- [Modrinth](https://modrinth.com) - Mod distribution platform
- [Charm](https://charm.sh) - TUI libraries (bubbletea, lipgloss)
- [Cobra](https://cobra.dev) - CLI framework
- [Podman](https://podman.io) - Rootless container runtime

---

**Made with ❤️ for the Minecraft community**

