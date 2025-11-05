# go-mc Development Guidelines

This document defines the development standards, workflow, and guidelines for contributing to **go-mc**. All contributors (human and AI) must follow these rules to maintain code quality, consistency, and project integrity.

---

## 📚 Table of Contents

1. [Philosophy](#philosophy)
2. [Sub-Agent Usage](#sub-agent-usage)
3. [Coding Standards](#coding-standards)
4. [Testing Requirements](#testing-requirements)
5. [Git Workflow](#git-workflow)
6. [Pull Request Process](#pull-request-process)
7. [Documentation Rules](#documentation-rules)
8. [CI/CD Behavior](#cicd-behavior)
9. [Roadmap Maintenance](#roadmap-maintenance)

---

## 🎯 Philosophy

### Omakase Principle

**"Chef's Choice"** - The tool should work perfectly out-of-the-box with sensible defaults. Advanced options exist but are never required.

**Design Decisions:**
- Default to Podman (rootless) over Docker
- YAML state files over databases
- Fabric-only (no vanilla/forge/paper)
- Modrinth-only (no CurseForge)
- Debian 12/13 only (no multi-distro complexity)

### Core Values

1. **Simplicity** - Code should be obvious, not clever
2. **Reliability** - Fail gracefully, never corrupt state
3. **Performance** - Optimize for common case, profile before optimizing
4. **Security** - Rootless by default, validate all inputs
5. **Testability** - Easy to test, high coverage

---

## 🤖 Sub-Agent Usage

### golang-pro Agent

**ALL Go development tasks MUST use the `golang-pro` sub-agent** located at `.claude/agents/golang-pro.md`.

**When to invoke:**
- Writing new Go code
- Refactoring existing code
- Implementing business logic
- Optimizing performance
- Reviewing code quality

**Invocation example:**
```
Please implement the server lifecycle service using the golang-pro agent.
```

The golang-pro agent ensures:
- Idiomatic Go patterns
- gofmt compliance
- golangci-lint passing
- Proper error handling
- Context propagation
- Race-free concurrency

---

## 💻 Coding Standards

### Go Version

- **Minimum:** Go 1.21+ (for `log/slog` stdlib)
- **Target:** Latest stable Go release

### Formatting

**Mandatory:**
- `gofmt` - All code must be formatted (enforced by pre-commit hook)
- `goimports` - Import ordering and unused import removal

**Run before commit:**
```bash
make fmt
```

### Linting

**golangci-lint Configuration:**

Run all linters:
```bash
make lint
```

**Enabled linters (non-exhaustive):**
- `errcheck` - Check error handling
- `gosec` - Security checks
- `govet` - Suspicious constructs
- `staticcheck` - Advanced static analysis
- `gocyclo` - Cyclomatic complexity
- `dupl` - Code duplication
- `goconst` - Repeated strings
- `misspell` - Spelling errors
- `unparam` - Unused parameters
- `unused` - Unused code

**Zero tolerance policy:** All linter warnings must be addressed before merge.

### Code Style

#### Project Structure

Follow standard Go project layout:
```
cmd/          - Entry points
internal/     - Private application code
pkg/          - Public libraries (if any)
test/         - Integration tests
```

#### Naming Conventions

**Packages:**
- Lowercase, single word
- No underscores or mixedCase
- Examples: `server`, `container`, `modrinth`

**Files:**
- Lowercase with underscores
- Examples: `server_service.go`, `lifecycle_test.go`

**Variables/Functions:**
- camelCase for private
- PascalCase for exported
- Meaningful names, avoid abbreviations

**Constants:**
- PascalCase for exported
- camelCase for private
- Group related constants in blocks

#### Idiomatic Patterns

**Interfaces:**
```go
// Accept interfaces, return structs
func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}

// Small, focused interfaces
type Reader interface {
    Read(ctx context.Context, id string) (*Server, error)
}
```

**Error Handling:**
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create server: %w", err)
}

// Custom error types for known conditions
type NotFoundError struct {
    Name string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("server %q not found", e.Name)
}
```

**Context Propagation:**
```go
// ALL blocking operations must accept context
func (s *Service) CreateServer(ctx context.Context, config *Config) error {
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Pass context to sub-operations
    if err := s.container.Create(ctx, config); err != nil {
        return err
    }

    return nil
}
```

**Configuration:**
```go
// Functional options pattern
type Option func(*Service)

func WithTimeout(d time.Duration) Option {
    return func(s *Service) {
        s.timeout = d
    }
}

func NewService(opts ...Option) *Service {
    s := &Service{timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

#### Concurrency

**Goroutine Management:**
```go
// Always use context for cancellation
ctx, cancel := context.WithCancel(ctx)
defer cancel()

go func() {
    select {
    case <-ctx.Done():
        return
    case result := <-ch:
        // Process result
    }
}()
```

**Channels:**
```go
// Channels for orchestration
resultCh := make(chan Result)
errCh := make(chan error, 1)

go func() {
    result, err := doWork()
    if err != nil {
        errCh <- err
        return
    }
    resultCh <- result
}()

select {
case result := <-resultCh:
    // Success
case err := <-errCh:
    // Error
case <-time.After(timeout):
    // Timeout
}
```

**Mutexes:**
```go
// Use sync.RWMutex for state protection
type Cache struct {
    mu    sync.RWMutex
    items map[string]Item
}

func (c *Cache) Get(key string) (Item, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, ok := c.items[key]
    return item, ok
}
```

#### YAML State Management

**Atomic Writes:**
```go
// Write to temp file, then rename (atomic)
func SaveState(path string, state *State) error {
    data, err := yaml.Marshal(state)
    if err != nil {
        return fmt.Errorf("marshal state: %w", err)
    }

    tmpFile := path + ".tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return fmt.Errorf("write temp file: %w", err)
    }

    if err := os.Rename(tmpFile, path); err != nil {
        return fmt.Errorf("rename temp file: %w", err)
    }

    return nil
}
```

**File Locking:**
```go
// Prevent concurrent access
func LockFile(path string) (*os.File, error) {
    f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        return nil, err
    }

    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        f.Close()
        return nil, err
    }

    return f, nil
}
```

#### JSON Output

**All non-interactive commands must support `--json` flag:**

```go
// Output structure
type Output struct {
    Status  string      `json:"status"`  // "success" or "error"
    Data    interface{} `json:"data,omitempty"`
    Message string      `json:"message,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// Usage
func printOutput(w io.Writer, jsonMode bool, data interface{}, msg string) error {
    if jsonMode {
        out := Output{
            Status:  "success",
            Data:    data,
            Message: msg,
        }
        enc := json.NewEncoder(w)
        enc.SetIndent("", "  ")
        return enc.Encode(out)
    }

    // Human-readable format
    fmt.Fprintln(w, msg)
    return nil
}
```

---

## 🧪 Testing Requirements

### Coverage Target

**Minimum: 70%** overall test coverage
**Goal: 80%+** for core business logic

**Check coverage:**
```bash
make coverage
```

### Test Types

#### Unit Tests

**Required for:**
- All service layer code
- State management functions
- Utility functions
- Business logic

**Pattern: Table-Driven Tests**
```go
func TestServerService_Create(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid config",
            config: &Config{
                Name:    "test",
                Memory:  "2G",
                Version: "1.20.4",
            },
            wantErr: false,
        },
        {
            name: "invalid memory",
            config: &Config{
                Name:    "test",
                Memory:  "invalid",
                Version: "1.20.4",
            },
            wantErr: true,
            errMsg:  "invalid memory format",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewService(mockRepo)
            err := svc.Create(context.Background(), tt.config)

            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

#### Integration Tests

**Scope: Happy Path Only**

Test critical user workflows:
- Create → Start → Stop → Remove server
- Mod installation with dependencies
- Backup and restore
- User management with UUID lookup

**Location:** `test/` directory

**Run integration tests:**
```bash
make test-integration
```

**Requirements:**
- Must run in isolated environment
- Clean up all resources after test
- Use test fixtures for predictable state
- Mock external APIs (Modrinth, Mojang)

**Example:**
```go
func TestServerLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup
    ctx := context.Background()
    client, err := container.NewClient()
    require.NoError(t, err)
    defer client.Close()

    // Test workflow
    t.Run("create server", func(t *testing.T) {
        config := &Config{
            Name:    "test-integration",
            Memory:  "1G",
            Version: "1.20.4",
        }

        err := service.Create(ctx, config)
        require.NoError(t, err)
    })

    t.Run("start server", func(t *testing.T) {
        err := service.Start(ctx, "test-integration")
        require.NoError(t, err)
    })

    // Cleanup
    defer func() {
        _ = service.Remove(ctx, "test-integration", true)
    }()
}
```

### Mocking

Use interfaces for testability:

```go
// Define interface
type ContainerClient interface {
    Create(ctx context.Context, config *ContainerConfig) (string, error)
    Start(ctx context.Context, id string) error
    Stop(ctx context.Context, id string) error
}

// Mock implementation
type mockContainerClient struct {
    mock.Mock
}

func (m *mockContainerClient) Create(ctx context.Context, config *ContainerConfig) (string, error) {
    args := m.Called(ctx, config)
    return args.String(0), args.Error(1)
}
```

### Benchmarks

**Required for:**
- Performance-critical code
- State file operations
- Container operations
- JSON parsing

**Example:**
```go
func BenchmarkStateLoad(b *testing.B) {
    path := "testdata/state.yaml"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := LoadState(path)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Race Detection

**All tests must pass with race detector:**
```bash
make test-race
```

---

## 🔀 Git Workflow

### Branch Naming

**Format:** `<type>/<issue-number>-<short-description>`

**Types:**
- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation only
- `test/` - Test additions/fixes
- `chore/` - Maintenance tasks

**Examples:**
```
feature/42-modrinth-integration
fix/123-yaml-corruption-on-crash
refactor/45-service-layer-cleanup
docs/67-update-readme-quickstart
```

### Commit Messages

**Format: Conventional Commits**

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting (no code change)
- `refactor` - Code restructuring
- `perf` - Performance improvement
- `test` - Test additions
- `chore` - Maintenance
- `ci` - CI/CD changes

**Examples:**
```
feat(servers): add Modrinth mod installation

Implements mod search and installation via Modrinth API with
automatic dependency resolution. Supports version matching for
Minecraft and Fabric versions.

Closes #42

---

fix(state): prevent YAML corruption on concurrent writes

Use atomic file writes (write to temp, then rename) to prevent
state file corruption when process is interrupted.

Fixes #123

---

docs(readme): update quick start guide

Add examples for mod installation and whitelist management.
```

**Rules:**
- Subject line ≤ 72 characters
- Use imperative mood ("add" not "added")
- No period at end of subject
- Body optional but recommended for complex changes
- Reference issues/PRs in footer

### Pre-commit Hooks

**Automatic checks before commit:**

```bash
# Install pre-commit hook
make install-hooks
```

**Hook runs:**
1. `gofmt` - Format code
2. `goimports` - Fix imports
3. `go vet` - Static analysis
4. `golangci-lint` - Full linting

**If hook fails, commit is blocked until fixed.**

---

## 🔍 Pull Request Process

### Before Opening PR

**Checklist:**
- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Coverage ≥ 70% (`make coverage`)
- [ ] Race detector passes (`make test-race`)
- [ ] Documentation updated (if needed)
- [ ] CHANGELOG.md updated (for user-facing changes)
- [ ] Roadmap updated (if completing phase/milestone)
- [ ] Commit messages follow conventional format
- [ ] Branch is up-to-date with main

### PR Title

**Format:** Same as commit message

```
feat(servers): add Modrinth mod installation
fix(state): prevent YAML corruption on concurrent writes
docs(readme): update quick start guide
```

### PR Description Template

```markdown
## Description
Brief description of changes.

## Motivation
Why is this change needed? What problem does it solve?

## Changes
- Bullet list of specific changes
- Include file/function names if helpful

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated (if applicable)
- [ ] Manual testing performed

## Screenshots (if UI/TUI changes)
[Add screenshots or terminal recordings]

## Checklist
- [ ] Tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Roadmap updated (if milestone completed)

## Related Issues
Closes #123
Relates to #456
```

### Review Process

1. **Automated Checks:** CI must pass (lint, test, security)
2. **Code Review:** At least 1 approval required
3. **Documentation Review:** Ensure docs are updated
4. **Roadmap Check:** Verify roadmap reflects progress

### PR Labels

**Auto-applied by CI:**
- `tests-passing` / `tests-failing`
- `lint-passing` / `lint-failing`
- `security-passing` / `security-issues`

**Manual labels:**
- `ready-for-review` - PR is ready for human review
- `work-in-progress` - Not ready for review
- `breaking-change` - Requires major version bump
- `needs-docs` - Documentation missing
- `needs-tests` - Test coverage insufficient

### Merge Strategy

**Squash and Merge:**
- Squash all commits into single commit
- Preserve PR title as commit message
- Include PR number in commit

**Example final commit:**
```
feat(servers): add Modrinth mod installation (#42)

Implements mod search and installation via Modrinth API with
automatic dependency resolution. Supports version matching for
Minecraft and Fabric versions.
```

---

## 📝 Documentation Rules

### README.md Updates

**README.md is the North Star document.**

**Update README.md when:**
- Adding new commands or features
- Changing command syntax or flags
- Updating architecture diagrams
- Modifying configuration options
- Changing system requirements

**Do NOT duplicate:**
- Task lists (use GitHub Issues)
- Detailed implementation notes (use inline code comments)

### Inline Code Documentation

**Required:**
- All exported functions/types
- Complex algorithms
- Non-obvious business logic
- Public interfaces

**Format:**
```go
// CreateServer creates a new Minecraft Fabric server with the given configuration.
// It pulls the container image if needed, allocates ports, creates volumes,
// and initializes the YAML state file.
//
// Returns an error if:
//   - The server name already exists
//   - Port allocation fails
//   - Container creation fails
//   - State file write fails
func CreateServer(ctx context.Context, config *ServerConfig) error {
    // Implementation
}
```

### CHANGELOG.md

**User-facing changes MUST be documented in CHANGELOG.md**

**Format: Keep a Changelog**

```markdown
## [Unreleased]

### Added
- Modrinth mod search and installation (#42)
- Global whitelist management (#67)

### Changed
- Improved error messages for port conflicts (#89)

### Fixed
- YAML corruption on concurrent writes (#123)

## [1.0.0] - 2025-01-20

### Added
- Initial release
- Server lifecycle management
- TUI dashboard
```

---

## 🚀 CI/CD Behavior

### Workflows

#### `.github/workflows/lint.yml`

**Triggers:**
- Every push to any branch
- All pull requests

**Jobs:**
1. `gofmt` check (fail if not formatted)
2. `golangci-lint` (fail on any warnings)
3. `go mod tidy` check (fail if not tidy)

**Fail → Block PR merge**

#### `.github/workflows/test.yml`

**Triggers:**
- Every push to any branch
- All pull requests

**Jobs:**
1. Unit tests (`go test ./...`)
2. Race detector (`go test -race ./...`)
3. Coverage report (fail if < 70%)
4. Integration tests (only on PR to main)

**Fail → Block PR merge**

#### `.github/workflows/security.yml`

**Triggers:**
- Every push to main
- Pull requests to main
- Weekly schedule (Monday 3 AM)

**Jobs:**
1. `gosec` - Security linting
2. `govulncheck` - Vulnerability scanning
3. `trivy` - Container image scanning
4. Dependency audit

**Fail → Block PR merge (critical/high only)**

#### `.github/workflows/release.yml`

**Triggers:**
- Push to `main` branch (after PR merge)
- Manual dispatch with version tag

**Jobs:**
1. Build binaries (linux/amd64, linux/arm64)
2. Create GitHub Release
3. Upload artifacts
4. Generate release notes from CHANGELOG.md
5. Update version badge

**Versioning:**
- Semantic Versioning (MAJOR.MINOR.PATCH)
- Auto-increment PATCH on main merge
- Manual MINOR/MAJOR via git tag

---

## 📅 Roadmap Maintenance

### **CRITICAL RULE: Roadmap Must Stay Current**

**Every PR that completes a roadmap task MUST update README.md roadmap section.**

### How to Update Roadmap

1. **Open README.md**
2. **Find relevant phase in Roadmap section**
3. **Mark completed items:**
   ```diff
   ### Phase 3: Server Lifecycle Commands
   - [x] `servers create` command
   - [x] `servers start/stop/restart` commands
   - [ ] `servers rm` command
   ```

4. **Update phase status if completed:**
   ```diff
   - ### Phase 3: Server Lifecycle Commands
   + ### Phase 3: Server Lifecycle Commands ✅
   ```

5. **Add new tasks if discovered during implementation:**
   ```diff
   ### Phase 3: Server Lifecycle Commands
   - [x] `servers create` command
   - [x] `servers start/stop/restart` commands
   - [ ] `servers rm` command
   + - [ ] Add confirmation prompt for rm command
   ```

### PR Review Checklist

**Reviewers MUST verify:**
- [ ] Roadmap reflects PR changes
- [ ] Completed checkboxes match implementation
- [ ] New tasks added if scope expanded
- [ ] Phase marked complete if all tasks done

### Example PR with Roadmap Update

```markdown
## Changes
- Implemented `servers create` command
- Added YAML state persistence
- Created port allocation logic

## Roadmap Updated
- [x] Phase 3: `servers create` command
- [x] Phase 3: Server state persistence in YAML

## Next Steps
Added new task to roadmap:
- [ ] Phase 3: Add `--dry-run` flag to create command
```

---

## 🔐 Security Considerations

### Input Validation

**ALWAYS validate:**
- Server names (alphanumeric + hyphen only)
- Memory sizes (parse and validate format)
- Port numbers (1-65535 range)
- File paths (no directory traversal)
- YAML content (validate structure)

### Secrets Management

**NEVER:**
- Log RCON passwords
- Commit .env files
- Print secrets in error messages

**DO:**
- Store RCON passwords in server YAML (consider encryption)
- Use environment variables for sensitive config
- Redact secrets in JSON output

### Container Security

**Requirements:**
- Run Podman rootless by default
- Drop unnecessary capabilities
- Use read-only root filesystem where possible
- Limit memory/CPU resources

---

## 🎓 Learning Resources

### Required Reading

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Recommended Tools

- [golangci-lint](https://golangci-lint.run/)
- [gotests](https://github.com/cweill/gotests) - Generate table tests
- [mockery](https://github.com/vektra/mockery) - Generate mocks
- [delve](https://github.com/go-delve/delve) - Debugger

---

## 📞 Questions?

- **Documentation:** Check README.md first
- **Issues:** Browse [GitHub Issues](https://github.com/steviee/go-mc/issues)
- **Discussions:** Use GitHub Discussions for questions
- **Bugs:** Open an issue with bug report template

---

**Remember: Quality over speed. Write code you'd be proud to maintain in 5 years.**

