# go-mc: Minecraft Server Management CLI

**Repository:** https://github.com/steviee/go-mc
**Language:** Go 1.21+
**Target Platform:** Debian 12/13 (Linux x86_64/ARM64)
**License:** MIT

---

## 📋 Table of Contents

1. [Vision & Objectives](#vision--objectives)
2. [CLI Commands Specification](#cli-commands-specification)
3. [Technical Architecture](#technical-architecture)
4. [Project Structure](#project-structure)
5. [Development Phases](#development-phases)
6. [Implementation Details](#implementation-details)
7. [Testing Strategy](#testing-strategy)
8. [Deployment & Distribution](#deployment--distribution)

---

## 🎯 Vision & Objectives

### Vision

Create a **professional, production-ready CLI tool** for managing Minecraft Java Edition servers on Debian systems using Docker containers. The tool provides a simple, intuitive interface similar to `docker`, `systemctl`, or `kubectl` with both tabular output and an interactive TUI dashboard.

### Core Principles

- **Zero Dependencies** - Single binary, no external runtime needed
- **Docker-Native** - Uses Docker for isolation and lifecycle management
- **State Tracking** - Local SQLite database tracks all servers per node
- **Sensible Defaults** - Works out-of-box, minimal configuration required
- **Unix Philosophy** - Does one thing well, composable with other tools
- **Interactive TUI** - Real-time monitoring with `go-mc top`

### Target Users

- **System Administrators** - Managing multiple Minecraft servers on dedicated hardware
- **Self-hosters** - Running game servers for friends/family
- **DevOps Engineers** - Infrastructure-as-Code for game server fleets

### Key Features

- Create Minecraft servers with smart defaults (version, memory, ports)
- Start/stop/restart lifecycle management
- Update Minecraft versions and mods
- Real-time monitoring with TUI dashboard (`top` command)
- Tabular server listing (`ps` command)
- Log streaming and searching
- Automatic cleanup of unused resources
- Backup and restore functionality
- RCON command execution
- Resource limits and health checks

---

## 🖥️ CLI Commands Specification

### Command Structure

```
go-mc [global-flags] <command> [command-flags] [arguments]
```

### Global Flags

```
--config, -c       Config file path (default: /etc/go-mc/config.yaml)
--state-dir, -s    State directory (default: /var/lib/go-mc)
--json             Output in JSON format
--quiet, -q        Suppress non-error output
--verbose, -v      Verbose logging
--help, -h         Show help
--version          Show version
```

---

### Core Commands

#### `go-mc create [NAME]`

Create a new Minecraft server instance.

**Flags:**
```
--version <version>          Minecraft version (default: latest stable)
--type <type>                Server type: vanilla, fabric, forge, paper, spigot (default: fabric)
--memory <size>              Memory allocation (default: 2G)
--port <port>                Server port (default: auto-assign from 25565+)
--rcon-port <port>           RCON port (default: auto-assign)
--rcon-password <password>   RCON password (default: auto-generated)
--world-seed <seed>          World seed
--difficulty <level>         Difficulty: peaceful, easy, normal, hard (default: normal)
--gamemode <mode>            Gamemode: survival, creative, adventure (default: survival)
--max-players <count>        Max players (default: 20)
--whitelist                  Enable whitelist (default: false)
--online-mode                Enable online mode (default: true)
--pvp                        Enable PVP (default: true)
--mods <url|path>            Comma-separated list of mod URLs or paths
--start                      Start server immediately after creation
--dry-run                    Show what would be created without doing it
```

**Examples:**
```bash
# Create with defaults
go-mc create survival-world

# Create Fabric server with specific version
go-mc create modded --version 1.20.4 --type fabric --memory 4G

# Create and start immediately
go-mc create test-server --start

# Create with mods
go-mc create modpack --type fabric --mods ./mods/,https://example.com/mod.jar
```

**Output:**
```
Creating Minecraft server 'survival-world'...
  Type:     fabric
  Version:  1.20.4
  Memory:   2G
  Port:     25565
  RCON:     25575 (password: abc123xyz)

Pulling Docker image itzg/minecraft-server:latest...
Creating container go-mc-survival-world...
Writing configuration to /var/lib/go-mc/servers/survival-world/

✓ Server 'survival-world' created successfully
  Container ID: a1b2c3d4e5f6

To start: go-mc start survival-world
To view:  go-mc logs survival-world -f
```

---

#### `go-mc ps`

List all servers in tabular format (like `docker ps`).

**Flags:**
```
--all, -a          Show all servers (including stopped)
--filter, -f       Filter by status: running, stopped, created, error
--format           Output format: table, json, yaml, wide
--no-header        Omit header row
--sort             Sort by: name, status, created, memory, cpu
```

**Output (default):**
```
NAME              STATUS      VERSION   PLAYERS   MEMORY    CPU      UPTIME
survival-world    running     1.20.4    3/20      1.8G/2G   12%      2d 5h
creative-test     running     1.20.1    0/10      800M/2G   2%       3h 24m
modded-server     stopped     1.19.4    -         -         -        -
build-world       running     1.20.4    1/20      1.2G/4G   8%       12h 3m
```

**Output (wide format):**
```
NAME            STATUS    VERSION   PLAYERS   MEMORY      CPU    PORT    CONTAINER-ID    CREATED
survival-world  running   1.20.4    3/20      1.8G/2G     12%    25565   a1b2c3d4e5f6    2025-01-15
creative-test   running   1.20.1    0/10      800M/2G     2%     25566   b2c3d4e5f6a1    2025-01-18
```

---

#### `go-mc top`

Interactive TUI dashboard (like `htop` or `lazydocker`).

**Features:**
- Real-time server metrics (CPU, memory, players, TPS)
- Color-coded status indicators
- Sortable columns
- Quick actions (start/stop/restart/logs) via keyboard shortcuts
- Auto-refresh (configurable interval)
- Resource graphs (memory/CPU over time)
- Event log panel

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
h/?           Show help
```

**Layout:**
```
╔═══════════════════════════════════════════════════════════════════════════╗
║ go-mc v1.0.0                    Node: debian-gameserver      Refresh: 1s  ║
╠═══════════════════════════════════════════════════════════════════════════╣
║ System   CPU: [████████░░] 42%   Memory: [██████░░░░] 12.4G/32G         ║
║ Docker   Containers: 4 running, 2 stopped   Images: 3   Volumes: 6       ║
╠═══════════════════════════════════════════════════════════════════════════╣
║ NAME              STATUS    VERSION  PLAYERS   MEM       CPU   TPS  PORT  ║
║ ▸ survival-world  ●running  1.20.4   3/20     1.8G/2G   12%   20   25565 ║
║   creative-test   ●running  1.20.1   0/10     800M/2G   2%    20   25566 ║
║   modded-server   ○stopped  1.19.4   -        -         -     -    25567 ║
║   build-world     ●running  1.20.4   1/20     1.2G/4G   8%    20   25568 ║
╠═══════════════════════════════════════════════════════════════════════════╣
║ Memory Usage (survival-world)        ┃ Recent Events                      ║
║ 2.0G ┤                               ┃ 14:32  survival-world started      ║
║ 1.5G ┤      ╭─╮                      ┃ 14:28  creative-test player join  ║
║ 1.0G ┤   ╭──╯ ╰────╮                 ┃ 14:15  modded-server stopped      ║
║ 0.5G ┤───╯          ╰────            ┃ 13:45  backup completed            ║
║      └─────────────────────────      ┃                                    ║
╠═══════════════════════════════════════════════════════════════════════════╣
║ s:start  x:stop  r:restart  l:logs  d:delete  u:update  q:quit  ?:help   ║
╚═══════════════════════════════════════════════════════════════════════════╝
```

---

#### `go-mc start [NAME...]`

Start one or more stopped servers.

**Flags:**
```
--all, -a          Start all stopped servers
--wait, -w         Wait until server is fully started
--timeout <dur>    Timeout for wait (default: 5m)
```

**Examples:**
```bash
go-mc start survival-world
go-mc start server1 server2 server3
go-mc start --all
go-mc start survival-world --wait
```

---

#### `go-mc stop [NAME...]`

Stop one or more running servers gracefully.

**Flags:**
```
--all, -a          Stop all running servers
--force, -f        Force stop (skip graceful shutdown)
--timeout <dur>    Graceful shutdown timeout (default: 30s)
```

**Examples:**
```bash
go-mc stop survival-world
go-mc stop --all
go-mc stop survival-world --force
```

---

#### `go-mc restart [NAME...]`

Restart one or more servers.

**Flags:**
```
--all, -a          Restart all servers
--wait, -w         Wait until restart is complete
```

---

#### `go-mc rm [NAME...]`

Remove one or more servers (with confirmation).

**Flags:**
```
--force, -f        Skip confirmation
--volumes, -v      Remove associated volumes (world data)
--keep-backups     Keep backups even if removing volumes
```

**Examples:**
```bash
go-mc rm old-server
go-mc rm test-server --force --volumes
```

---

#### `go-mc logs [NAME]`

View server logs.

**Flags:**
```
--follow, -f       Follow log output (stream)
--tail <n>         Show last N lines (default: 100)
--since <time>     Show logs since timestamp/duration
--timestamps, -t   Show timestamps
--grep <pattern>   Filter logs by pattern (regex)
```

**Examples:**
```bash
go-mc logs survival-world
go-mc logs survival-world -f
go-mc logs survival-world --tail 50 --grep "ERROR"
go-mc logs survival-world --since 1h
```

---

#### `go-mc exec [NAME] <command>`

Execute RCON command on server.

**Examples:**
```bash
go-mc exec survival-world "say Hello players!"
go-mc exec survival-world "whitelist add Steve"
go-mc exec survival-world "gamemode creative Alex"
go-mc exec survival-world "save-all"
```

---

#### `go-mc update [NAME]`

Update Minecraft server version or mods.

**Flags:**
```
--version <version>    Update to specific version
--latest               Update to latest version
--mods                 Update mods only
--backup               Create backup before update (default: true)
--restart              Restart after update (default: false)
```

**Examples:**
```bash
go-mc update survival-world --version 1.20.5
go-mc update survival-world --latest --restart
go-mc update modded-server --mods
```

---

#### `go-mc inspect [NAME]`

Show detailed information about a server.

**Output:**
```yaml
Name: survival-world
Status: running
Created: 2025-01-15 10:30:45
Uptime: 2 days 5 hours 23 minutes

Container:
  ID: a1b2c3d4e5f6789
  Image: itzg/minecraft-server:latest
  Platform: linux/amd64

Configuration:
  Type: fabric
  Version: 1.20.4
  Gamemode: survival
  Difficulty: normal
  Max Players: 20
  Online Mode: true
  PVP: true
  Whitelist: false

Network:
  Server Port: 25565
  RCON Port: 25575
  RCON Password: abc123xyz

Resources:
  Memory Limit: 2G
  Memory Usage: 1.8G (90%)
  CPU Usage: 12%

Players Online: 3/20
  - Steve (joined 2h ago)
  - Alex (joined 45m ago)
  - Notch (joined 12m ago)

Volumes:
  - /var/lib/go-mc/servers/survival-world/data
  - /var/lib/go-mc/servers/survival-world/mods

Mods: 15
  - fabric-api-0.92.0+1.20.4.jar
  - sodium-fabric-mc1.20.4-0.5.5.jar
  - lithium-fabric-mc1.20.4-0.12.0.jar
  ...

Backups: 5 (latest: 2025-01-18 03:00:00)
```

---

#### `go-mc backup [NAME]`

Create backup of server data.

**Flags:**
```
--all, -a          Backup all servers
--compress         Compress backup (default: true)
--output, -o       Output directory (default: /var/lib/go-mc/backups)
--keep <n>         Keep last N backups (default: 5)
```

---

#### `go-mc restore [NAME] <backup-id>`

Restore server from backup.

**Flags:**
```
--force, -f        Overwrite existing data
--stop             Stop server before restore (default: true)
```

---

#### `go-mc cleanup`

Clean up unused Docker resources.

**Flags:**
```
--all, -a          Clean all (images, volumes, networks)
--images           Clean unused images
--volumes          Clean unused volumes
--force, -f        Skip confirmation
--dry-run          Show what would be cleaned
```

---

#### `go-mc config`

Manage global configuration.

**Subcommands:**
```bash
go-mc config get <key>
go-mc config set <key> <value>
go-mc config list
go-mc config reset
```

---

#### `go-mc version`

Show version information.

**Output:**
```
go-mc version v1.0.0
  Build:    2025-01-20 14:32:15
  Commit:   a1b2c3d
  Go:       1.21.5
  OS/Arch:  linux/amd64
  Docker:   24.0.7 (API 1.43)
```

---

## 🏗️ Technical Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        go-mc CLI                            │
│                    (Single Binary)                          │
├─────────────────────────────────────────────────────────────┤
│  CLI Layer        (cobra + flags parsing)                   │
│  TUI Layer        (bubbletea + lipgloss)                    │
│  Business Logic   (service layer)                           │
│  State Manager    (SQLite database)                         │
│  Docker Client    (Docker SDK for Go)                       │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Docker Engine                            │
├─────────────────────────────────────────────────────────────┤
│  Container: go-mc-survival-world                            │
│  Container: go-mc-creative-test                             │
│  Container: go-mc-modded-server                             │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                Local State & Data                           │
├─────────────────────────────────────────────────────────────┤
│  /var/lib/go-mc/                                            │
│    ├── state.db              (SQLite state tracking)        │
│    ├── config.yaml           (global config)                │
│    ├── servers/                                             │
│    │   ├── survival-world/                                  │
│    │   │   ├── data/         (world, config, logs)         │
│    │   │   └── mods/         (mod files)                    │
│    │   └── creative-test/                                   │
│    └── backups/              (compressed backups)           │
└─────────────────────────────────────────────────────────────┘
```

---

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Language** | Go 1.21+ | Core application |
| **CLI Framework** | [cobra](https://github.com/spf13/cobra) | Command structure & parsing |
| **TUI Framework** | [bubbletea](https://github.com/charmbracelet/bubbletea) | Interactive terminal UI |
| **TUI Styling** | [lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| **Tables** | [bubbles/table](https://github.com/charmbracelet/bubbles) | Table rendering |
| **Charts** | [termui](https://github.com/gizak/termui) or custom | Resource graphs |
| **Docker Client** | [docker/docker](https://github.com/docker/docker) | Container management |
| **Database** | [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | Pure-Go SQLite |
| **Config** | [viper](https://github.com/spf13/viper) | Config management |
| **Logging** | [slog](https://pkg.go.dev/log/slog) | Structured logging (stdlib) |
| **RCON** | [gorcon/rcon](https://github.com/gorcon/rcon-cli) | Minecraft RCON protocol |
| **HTTP Client** | [net/http](https://pkg.go.dev/net/http) | Download mods/versions |
| **Testing** | [testify](https://github.com/stretchr/testify) | Test assertions |

---

### State Management

#### SQLite Schema

```sql
-- servers table: tracks all Minecraft server instances
CREATE TABLE servers (
    id TEXT PRIMARY KEY,                    -- Unique identifier (UUID)
    name TEXT UNIQUE NOT NULL,              -- Server name (used in container name)
    status TEXT NOT NULL,                   -- running, stopped, created, error
    container_id TEXT,                      -- Docker container ID

    -- Minecraft configuration
    mc_version TEXT NOT NULL,               -- 1.20.4
    mc_type TEXT NOT NULL,                  -- vanilla, fabric, forge, paper, spigot

    -- Resource limits
    memory_limit TEXT NOT NULL,             -- 2G, 4G, etc.
    cpu_limit REAL,                         -- CPU shares (optional)

    -- Network configuration
    server_port INTEGER NOT NULL,           -- 25565
    rcon_port INTEGER NOT NULL,             -- 25575
    rcon_password TEXT NOT NULL,            -- encrypted

    -- Game configuration
    difficulty TEXT,                        -- peaceful, easy, normal, hard
    gamemode TEXT,                          -- survival, creative, adventure
    max_players INTEGER,
    whitelist_enabled BOOLEAN,
    online_mode BOOLEAN,
    pvp_enabled BOOLEAN,
    world_seed TEXT,

    -- Paths
    data_path TEXT NOT NULL,                -- /var/lib/go-mc/servers/{name}/data
    mods_path TEXT,                         -- /var/lib/go-mc/servers/{name}/mods

    -- Timestamps
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    started_at DATETIME,
    stopped_at DATETIME,

    -- Metadata
    tags TEXT,                              -- JSON array of tags
    notes TEXT                              -- User notes
);

-- server_mods table: tracks installed mods per server
CREATE TABLE server_mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    url TEXT,                               -- Source URL (if downloaded)
    version TEXT,
    sha256 TEXT,                            -- File hash for verification
    installed_at DATETIME NOT NULL,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
);

-- backups table: tracks server backups
CREATE TABLE backups (
    id TEXT PRIMARY KEY,                    -- UUID
    server_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    compressed BOOLEAN NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
);

-- events table: audit log of all operations
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL,               -- create, start, stop, delete, update, etc.
    server_id TEXT,
    details TEXT,                           -- JSON details
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE SET NULL
);

-- metrics table: historical metrics for charting (optional)
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    server_id TEXT NOT NULL,
    cpu_percent REAL,
    memory_bytes INTEGER,
    players_online INTEGER,
    tps REAL,                               -- Ticks per second
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_name ON servers(name);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_server_id ON events(server_id);
CREATE INDEX idx_metrics_server_timestamp ON metrics(server_id, timestamp);
```

---

### Configuration File

**Location:** `/etc/go-mc/config.yaml` (or `~/.config/go-mc/config.yaml`)

```yaml
# go-mc configuration

# Global settings
state_dir: /var/lib/go-mc
log_level: info
log_file: /var/log/go-mc/go-mc.log

# Docker settings
docker:
  socket: /var/run/docker.sock
  network: go-mc-network
  image: itzg/minecraft-server:latest
  auto_pull: true

# Default server settings
defaults:
  memory: 2G
  type: fabric
  version: latest
  difficulty: normal
  gamemode: survival
  max_players: 20
  online_mode: true
  pvp: true
  whitelist: false

  # Port allocation strategy
  port_start: 25565
  rcon_port_start: 25575

  # Auto-generated RCON password length
  rcon_password_length: 16

# Backup settings
backups:
  directory: /var/lib/go-mc/backups
  compress: true
  keep_count: 5
  schedule: "0 3 * * *"  # Daily at 3 AM (if daemon mode)

# TUI settings
tui:
  refresh_interval: 1s
  theme: default  # default, dark, light, custom
  colors:
    running: green
    stopped: gray
    error: red

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

---

## 📁 Project Structure

```
go-mc/
├── cmd/
│   └── go-mc/
│       └── main.go                    # Entry point
│
├── internal/
│   ├── cli/                           # CLI commands
│   │   ├── root.go                    # Root command & global flags
│   │   ├── create.go                  # Create command
│   │   ├── ps.go                      # List command
│   │   ├── top.go                     # TUI dashboard command
│   │   ├── start.go                   # Start command
│   │   ├── stop.go                    # Stop command
│   │   ├── restart.go                 # Restart command
│   │   ├── rm.go                      # Remove command
│   │   ├── logs.go                    # Logs command
│   │   ├── exec.go                    # RCON exec command
│   │   ├── update.go                  # Update command
│   │   ├── inspect.go                 # Inspect command
│   │   ├── backup.go                  # Backup command
│   │   ├── restore.go                 # Restore command
│   │   ├── cleanup.go                 # Cleanup command
│   │   ├── config.go                  # Config management
│   │   └── version.go                 # Version command
│   │
│   ├── tui/                           # Terminal UI components
│   │   ├── dashboard.go               # Main dashboard (top command)
│   │   ├── models.go                  # Bubbletea models
│   │   ├── views.go                   # View rendering
│   │   ├── styles.go                  # Lipgloss styles
│   │   ├── table.go                   # Server list table
│   │   ├── chart.go                   # Resource charts
│   │   └── events.go                  # Event log panel
│   │
│   ├── service/                       # Business logic
│   │   ├── server_service.go          # Server CRUD operations
│   │   ├── lifecycle_service.go       # Start/stop/restart logic
│   │   ├── docker_service.go          # Docker API wrapper
│   │   ├── rcon_service.go            # RCON client wrapper
│   │   ├── backup_service.go          # Backup/restore logic
│   │   ├── update_service.go          # Version update logic
│   │   ├── metrics_service.go         # Metrics collection
│   │   └── cleanup_service.go         # Resource cleanup
│   │
│   ├── state/                         # State management
│   │   ├── db.go                      # Database connection & migrations
│   │   ├── server_repo.go             # Server repository
│   │   ├── mod_repo.go                # Mod repository
│   │   ├── backup_repo.go             # Backup repository
│   │   ├── event_repo.go              # Event repository
│   │   └── metrics_repo.go            # Metrics repository
│   │
│   ├── docker/                        # Docker integration
│   │   ├── client.go                  # Docker client wrapper
│   │   ├── container.go               # Container operations
│   │   ├── image.go                   # Image operations
│   │   ├── network.go                 # Network operations
│   │   ├── volume.go                  # Volume operations
│   │   └── stats.go                   # Stats streaming
│   │
│   ├── minecraft/                     # Minecraft-specific logic
│   │   ├── version.go                 # Version management
│   │   ├── rcon.go                    # RCON protocol
│   │   ├── properties.go              # server.properties parser
│   │   ├── whitelist.go               # whitelist.json management
│   │   ├── ops.go                     # ops.json management
│   │   └── mods.go                    # Mod management
│   │
│   ├── config/                        # Configuration
│   │   ├── config.go                  # Config struct & loading
│   │   ├── defaults.go                # Default values
│   │   └── validation.go              # Config validation
│   │
│   ├── util/                          # Utilities
│   │   ├── logger.go                  # Logging setup
│   │   ├── format.go                  # Output formatting
│   │   ├── ports.go                   # Port allocation
│   │   ├── fs.go                      # Filesystem helpers
│   │   ├── archive.go                 # Tar/gzip helpers
│   │   └── password.go                # Password generation
│   │
│   └── models/                        # Data models
│       ├── server.go                  # Server model
│       ├── backup.go                  # Backup model
│       ├── event.go                   # Event model
│       └── metrics.go                 # Metrics model
│
├── pkg/                               # Public API (if needed for plugins)
│   └── api/
│       └── server.go
│
├── test/                              # Integration tests
│   ├── create_test.go
│   ├── lifecycle_test.go
│   └── backup_test.go
│
├── scripts/                           # Build & deployment scripts
│   ├── build.sh                       # Build script
│   ├── install.sh                     # Installation script
│   └── uninstall.sh                   # Uninstallation script
│
├── docs/                              # Documentation
│   ├── README.md                      # Getting started
│   ├── ARCHITECTURE.md                # Architecture details
│   ├── COMMANDS.md                    # Command reference
│   └── DEVELOPMENT.md                 # Development guide
│
├── .github/
│   └── workflows/
│       ├── build.yml                  # Build workflow
│       ├── test.yml                   # Test workflow
│       └── release.yml                # Release workflow
│
├── go.mod                             # Go module definition
├── go.sum                             # Go dependencies lock
├── Makefile                           # Build automation
├── README.md                          # Project README
├── LICENSE                            # MIT License
└── .gitignore
```

---

## 🚀 Development Phases

### **Phase 0: Project Setup** (1-2 days)

**Goal:** Initialize Go project with basic structure and dependencies.

#### Tasks

- [ ] Initialize Git repository at `github.com/steviee/go-mc`
- [ ] Create Go module: `go mod init github.com/steviee/go-mc`
- [ ] Set up directory structure (internal/, cmd/, pkg/, docs/)
- [ ] Add dependencies (cobra, bubbletea, docker SDK, sqlite)
- [ ] Create Makefile for build automation
- [ ] Set up GitHub Actions for CI/CD
- [ ] Write initial README.md

#### Acceptance Criteria

- Project compiles: `go build ./cmd/go-mc`
- Tests run: `go test ./...`
- Basic `go-mc --version` works

---

### **Phase 1: Core CLI & State Management** (3-4 days)

**Goal:** Implement basic CLI structure and state tracking.

#### Tasks

- [ ] **CLI Framework**
  - [ ] Set up cobra root command with global flags
  - [ ] Implement `version` command
  - [ ] Implement `config` command (get/set/list)
  - [ ] Add structured logging with slog

- [ ] **State Management**
  - [ ] Design SQLite schema (servers, mods, backups, events)
  - [ ] Implement database initialization & migrations
  - [ ] Create repository layer (server_repo, event_repo)
  - [ ] Write tests for repositories

- [ ] **Configuration**
  - [ ] Load config from YAML file
  - [ ] Support config precedence (file → env → flags)
  - [ ] Implement sensible defaults
  - [ ] Add config validation

#### Acceptance Criteria

- `go-mc version` shows version info
- `go-mc config list` shows current config
- SQLite database created at `/var/lib/go-mc/state.db`
- All config tests pass

---

### **Phase 2: Docker Integration** (3-4 days)

**Goal:** Implement Docker client wrapper and container lifecycle.

#### Tasks

- [ ] **Docker Client**
  - [ ] Initialize Docker SDK client
  - [ ] Implement connection test & version check
  - [ ] Handle Docker daemon not running error

- [ ] **Container Operations**
  - [ ] Create container with minecraft-server image
  - [ ] Start/stop/restart containers
  - [ ] Remove containers
  - [ ] Stream container logs
  - [ ] Get container stats (CPU, memory)

- [ ] **Network & Volumes**
  - [ ] Create Docker network (`go-mc-network`)
  - [ ] Create named volumes for server data
  - [ ] Implement port allocation strategy
  - [ ] Handle port conflicts

- [ ] **Testing**
  - [ ] Unit tests for Docker wrapper
  - [ ] Integration tests with real Docker

#### Acceptance Criteria

- Can connect to Docker daemon
- Can create/start/stop/remove test containers
- Ports auto-allocated without conflicts
- Container stats retrieved successfully

---

### **Phase 3: Server Lifecycle Commands** (4-5 days)

**Goal:** Implement core server management commands.

#### Tasks

- [ ] **Create Command** (`go-mc create`)
  - [ ] Parse all flags (version, type, memory, ports, etc.)
  - [ ] Validate inputs (memory format, version exists, port range)
  - [ ] Generate server configuration
  - [ ] Pull Docker image if needed
  - [ ] Create container with environment variables
  - [ ] Create directories and volumes
  - [ ] Store server in database
  - [ ] Log event
  - [ ] Show success message with details

- [ ] **Start/Stop/Restart Commands**
  - [ ] `go-mc start` - Start containers
  - [ ] `go-mc stop` - Graceful shutdown with timeout
  - [ ] `go-mc restart` - Stop then start
  - [ ] Support multiple servers
  - [ ] Update database status
  - [ ] Handle errors gracefully

- [ ] **Remove Command** (`go-mc rm`)
  - [ ] Confirmation prompt (skip with --force)
  - [ ] Stop container if running
  - [ ] Remove container
  - [ ] Optionally remove volumes (--volumes)
  - [ ] Remove from database
  - [ ] Log event

- [ ] **List Command** (`go-mc ps`)
  - [ ] Query database for servers
  - [ ] Fetch Docker stats for running servers
  - [ ] Format as table (use tablewriter)
  - [ ] Support filtering (--filter status=running)
  - [ ] Support sorting (--sort name)
  - [ ] Support JSON output (--json)

#### Acceptance Criteria

- Can create server: `go-mc create test-server`
- Can start server: `go-mc start test-server`
- Can stop server: `go-mc stop test-server`
- Can list servers: `go-mc ps`
- Can remove server: `go-mc rm test-server --force`
- All operations logged to database

---

### **Phase 4: Logs & Inspect** (2-3 days)

**Goal:** Implement log viewing and detailed inspection.

#### Tasks

- [ ] **Logs Command** (`go-mc logs`)
  - [ ] Stream container logs (--follow)
  - [ ] Show last N lines (--tail 100)
  - [ ] Filter logs with regex (--grep)
  - [ ] Show logs since timestamp (--since 1h)
  - [ ] Add timestamps option
  - [ ] Handle log rotation

- [ ] **Inspect Command** (`go-mc inspect`)
  - [ ] Query server from database
  - [ ] Get container details from Docker
  - [ ] Parse server.properties
  - [ ] List installed mods
  - [ ] Show player list (via RCON)
  - [ ] Show backup history
  - [ ] Format as YAML or JSON

#### Acceptance Criteria

- `go-mc logs server -f` streams logs
- `go-mc logs server --tail 50 --grep ERROR` works
- `go-mc inspect server` shows full details

---

### **Phase 5: RCON Integration** (2-3 days)

**Goal:** Implement RCON command execution.

#### Tasks

- [ ] **RCON Client**
  - [ ] Implement RCON protocol (or use library)
  - [ ] Handle authentication
  - [ ] Send commands
  - [ ] Parse responses

- [ ] **Exec Command** (`go-mc exec`)
  - [ ] Get RCON credentials from database
  - [ ] Connect to server
  - [ ] Execute command
  - [ ] Print response
  - [ ] Handle errors (server offline, auth failed)

- [ ] **Integration with Other Commands**
  - [ ] Get player count for `ps` command
  - [ ] Get TPS for metrics
  - [ ] Implement graceful stop with `/stop` command

#### Acceptance Criteria

- `go-mc exec server "say Hello"` executes command
- `go-mc exec server "list"` shows players
- RCON errors handled gracefully

---

### **Phase 6: TUI Dashboard** (5-7 days)

**Goal:** Implement interactive `go-mc top` dashboard.

#### Tasks

- [ ] **Bubbletea Setup**
  - [ ] Create main model struct
  - [ ] Implement Init/Update/View methods
  - [ ] Handle keyboard input
  - [ ] Set up auto-refresh ticker

- [ ] **UI Components**
  - [ ] Header with system info
  - [ ] Server list table (sortable, navigable)
  - [ ] Status indicators (colored bullets)
  - [ ] Memory/CPU usage graphs (sparklines)
  - [ ] Event log panel
  - [ ] Footer with keyboard shortcuts

- [ ] **Styling**
  - [ ] Define color scheme with lipgloss
  - [ ] Create reusable styles
  - [ ] Handle terminal resize
  - [ ] Support dark/light themes

- [ ] **Interactive Actions**
  - [ ] Navigate servers with arrow keys
  - [ ] Start/stop/restart with hotkeys
  - [ ] View logs (open in less/tail)
  - [ ] Show details panel
  - [ ] Delete with confirmation

- [ ] **Real-time Updates**
  - [ ] Fetch server list every N seconds
  - [ ] Update Docker stats
  - [ ] Poll RCON for player count/TPS
  - [ ] Show recent events from database

#### Acceptance Criteria

- `go-mc top` launches TUI dashboard
- Can navigate servers with keyboard
- Can start/stop servers from TUI
- Metrics update in real-time
- TUI handles terminal resize

---

### **Phase 7: Update & Backup** (4-5 days)

**Goal:** Implement version updates and backup/restore.

#### Tasks

- [ ] **Version Management**
  - [ ] Fetch available Minecraft versions from Mojang API
  - [ ] Validate version format
  - [ ] Detect server type (vanilla/fabric/forge/paper)

- [ ] **Update Command** (`go-mc update`)
  - [ ] Create backup before update (safety)
  - [ ] Stop server
  - [ ] Update Docker image or environment variables
  - [ ] Update mods (if specified)
  - [ ] Start server (if --restart)
  - [ ] Verify update success
  - [ ] Rollback on failure

- [ ] **Backup Command** (`go-mc backup`)
  - [ ] Stop server or use `/save-all` + `/save-off` (RCON)
  - [ ] Create tar.gz of world data
  - [ ] Store backup metadata in database
  - [ ] Resume server
  - [ ] Implement backup rotation (keep last N)

- [ ] **Restore Command** (`go-mc restore`)
  - [ ] List available backups
  - [ ] Stop server
  - [ ] Extract backup to data directory
  - [ ] Update database
  - [ ] Restart server

#### Acceptance Criteria

- `go-mc update server --version 1.20.5` updates version
- `go-mc backup server` creates backup
- `go-mc restore server <backup-id>` restores backup
- Backups rotated automatically

---

### **Phase 8: Cleanup & Maintenance** (2-3 days)

**Goal:** Implement resource cleanup and maintenance tasks.

#### Tasks

- [ ] **Cleanup Command** (`go-mc cleanup`)
  - [ ] Find unused Docker images
  - [ ] Find unused volumes
  - [ ] Find old backups (> 180 days)
  - [ ] Show dry-run preview
  - [ ] Execute cleanup with confirmation
  - [ ] Log cleanup actions

- [ ] **Maintenance Utilities**
  - [ ] Disk usage report
  - [ ] Health check command
  - [ ] Validate database integrity

#### Acceptance Criteria

- `go-mc cleanup --dry-run` shows what would be cleaned
- `go-mc cleanup --force` cleans resources
- Disk space reclaimed

---

### **Phase 9: Polish & Documentation** (3-4 days)

**Goal:** Finalize features, testing, and documentation.

#### Tasks

- [ ] **Error Handling**
  - [ ] Improve error messages
  - [ ] Add suggestions for common errors
  - [ ] Handle edge cases

- [ ] **Testing**
  - [ ] Write unit tests for all services
  - [ ] Write integration tests
  - [ ] Test on Debian 12/13
  - [ ] Test with various Docker versions

- [ ] **Documentation**
  - [ ] Write comprehensive README
  - [ ] Create command reference (COMMANDS.md)
  - [ ] Write architecture documentation
  - [ ] Add examples and tutorials
  - [ ] Create troubleshooting guide

- [ ] **Build & Distribution**
  - [ ] Create build script
  - [ ] Set up GitHub Releases
  - [ ] Build for multiple architectures (amd64, arm64)
  - [ ] Create installation script
  - [ ] Package as .deb (optional)

#### Acceptance Criteria

- All tests pass on Debian 12/13
- Documentation complete
- Binary builds for amd64 and arm64
- GitHub Release created

---

## 🔧 Implementation Details

### Smart Defaults

When creating a server without specifying all options:

```bash
go-mc create my-server
```

The system should:

1. **Minecraft Version:** Use latest stable release (fetch from Mojang API)
2. **Server Type:** Use Fabric (best for mods)
3. **Memory:** Allocate 2G (good for small servers)
4. **Port:** Auto-assign starting from 25565 (check available)
5. **RCON Port:** Auto-assign starting from 25575
6. **RCON Password:** Generate secure random password
7. **Difficulty:** Normal (balanced gameplay)
8. **Gamemode:** Survival (default Minecraft experience)
9. **Max Players:** 20 (reasonable default)
10. **World Seed:** Random (new world each time)

### Port Allocation Strategy

```go
// Pseudo-code
func AllocatePort(startPort int) (int, error) {
    usedPorts := GetAllUsedPorts() // from database

    for port := startPort; port < startPort + 1000; port++ {
        if !contains(usedPorts, port) && IsPortAvailable(port) {
            return port, nil
        }
    }

    return 0, errors.New("no available ports")
}
```

### Container Naming Convention

```
go-mc-<server-name>
```

Examples:
- `go-mc-survival-world`
- `go-mc-creative-test`
- `go-mc-modded-server`

### Docker Container Configuration

```go
containerConfig := &container.Config{
    Image: "itzg/minecraft-server:latest",
    Env: []string{
        "EULA=TRUE",
        fmt.Sprintf("VERSION=%s", version),
        fmt.Sprintf("TYPE=%s", serverType), // VANILLA, FABRIC, FORGE, PAPER, SPIGOT
        fmt.Sprintf("MEMORY=%s", memory),
        fmt.Sprintf("DIFFICULTY=%s", difficulty),
        fmt.Sprintf("MODE=%s", gamemode),
        fmt.Sprintf("MAX_PLAYERS=%d", maxPlayers),
        fmt.Sprintf("ENABLE_RCON=true"),
        fmt.Sprintf("RCON_PASSWORD=%s", rconPassword),
        fmt.Sprintf("RCON_PORT=%d", rconPort),
        fmt.Sprintf("WHITELIST=%t", whitelistEnabled),
        fmt.Sprintf("ONLINE_MODE=%t", onlineMode),
        fmt.Sprintf("PVP=%t", pvpEnabled),
    },
}

hostConfig := &container.HostConfig{
    PortBindings: nat.PortMap{
        nat.Port(fmt.Sprintf("%d/tcp", 25565)): []nat.PortBinding{
            {HostPort: fmt.Sprintf("%d", serverPort)},
        },
        nat.Port(fmt.Sprintf("%d/tcp", rconPort)): []nat.PortBinding{
            {HostPort: fmt.Sprintf("%d", rconPort)},
        },
    },
    Mounts: []mount.Mount{
        {
            Type:   mount.TypeBind,
            Source: dataPath,
            Target: "/data",
        },
        {
            Type:   mount.TypeBind,
            Source: modsPath,
            Target: "/mods",
        },
    },
    RestartPolicy: container.RestartPolicy{
        Name: "unless-stopped",
    },
    Resources: container.Resources{
        Memory: ParseMemory(memory), // Convert "2G" to bytes
    },
}
```

---

## 🧪 Testing Strategy

### Unit Tests

- **Repository Layer:** Test database operations with in-memory SQLite
- **Service Layer:** Mock Docker client, test business logic
- **CLI Commands:** Test flag parsing and validation
- **Utilities:** Test port allocation, password generation, formatting

### Integration Tests

- **Docker Integration:** Test against real Docker daemon (in CI)
- **End-to-End:** Create → Start → Stop → Remove full lifecycle
- **RCON:** Test command execution against real server

### Manual Testing Checklist

- [ ] Install on fresh Debian 12
- [ ] Create server with defaults
- [ ] Create server with all flags
- [ ] Start/stop/restart multiple servers
- [ ] Test port conflicts
- [ ] Test memory limits
- [ ] View logs (follow mode)
- [ ] Execute RCON commands
- [ ] Launch TUI dashboard
- [ ] Create and restore backups
- [ ] Update server version
- [ ] Cleanup unused resources
- [ ] Test with network disconnected
- [ ] Test with Docker daemon stopped

---

## 📦 Deployment & Distribution

### Build Process

```bash
# Build for local architecture
make build

# Build for all architectures
make build-all

# Builds:
# - build/go-mc-linux-amd64
# - build/go-mc-linux-arm64
```

### Installation Script

```bash
#!/bin/bash
# install.sh

VERSION="v1.0.0"
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

BINARY_URL="https://github.com/steviee/go-mc/releases/download/${VERSION}/go-mc-${OS}-${ARCH}"

# Download binary
curl -L "$BINARY_URL" -o /usr/local/bin/go-mc
chmod +x /usr/local/bin/go-mc

# Create directories
mkdir -p /var/lib/go-mc
mkdir -p /etc/go-mc
mkdir -p /var/log/go-mc

# Generate default config
go-mc config init

echo "go-mc installed successfully!"
echo "Run 'go-mc version' to verify installation."
```

### Package Distribution

1. **GitHub Releases:** Precompiled binaries for Linux amd64/arm64
2. **.deb Package (optional):** For apt-based systems
3. **Docker Image (future):** Run go-mc itself in container

---

## 🎯 Success Criteria

### Minimal Viable Product (MVP)

- [ ] Single binary that runs on Debian 12/13
- [ ] `create`, `start`, `stop`, `rm`, `ps` commands work
- [ ] State persisted in SQLite
- [ ] Basic error handling and logging
- [ ] Documentation (README + command help)

### Version 1.0

- [ ] All commands implemented
- [ ] TUI dashboard fully functional
- [ ] Backup/restore working
- [ ] Update mechanism working
- [ ] Comprehensive tests (>80% coverage)
- [ ] Full documentation
- [ ] GitHub Releases with binaries

### Future Enhancements (v1.1+)

- [ ] Multi-node support (manage servers across multiple machines)
- [ ] Web API (optional REST API for integration)
- [ ] Plugins system (extend with custom commands)
- [ ] Scheduled tasks (auto-backup, auto-update)
- [ ] Metrics export (Prometheus exporter)
- [ ] Notification system (Discord, email alerts)

---

## 📚 References

- **Go Docker SDK:** https://pkg.go.dev/github.com/docker/docker
- **Cobra CLI:** https://github.com/spf13/cobra
- **Bubbletea TUI:** https://github.com/charmbracelet/bubbletea
- **itzg Minecraft Docker:** https://github.com/itzg/docker-minecraft-server
- **Minecraft RCON:** https://wiki.vg/RCON
- **Mojang Version Manifest:** https://launchermeta.mojang.com/mc/game/version_manifest.json

---

## 📝 Notes

- **Go Version:** Minimum Go 1.21 for `log/slog` stdlib
- **Debian Compatibility:** Test on Debian 12 (Bookworm) and 13 (Trixie)
- **Docker Version:** Require Docker 20.10+ (API version 1.41+)
- **Root Requirements:** Some operations may require root/sudo for port binding <1024
- **Concurrency:** Use goroutines for background tasks (metrics collection, log streaming)
- **Graceful Shutdown:** Handle SIGINT/SIGTERM to clean up resources

---

**End of Project Plan**

This document serves as the comprehensive blueprint for the `go-mc` project. All development should reference this plan and update it as decisions evolve.
