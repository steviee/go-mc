package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/steviee/go-mc/internal/container"
	"github.com/steviee/go-mc/internal/state"
)

// OperationResult aggregates results from multiple server operations
type OperationResult struct {
	Success []string          // Successfully processed servers
	Failed  map[string]string // Failed servers with error messages
	Skipped []string          // Skipped servers (already in target state)
}

// NewOperationResult creates a new operation result
func NewOperationResult() *OperationResult {
	return &OperationResult{
		Success: make([]string, 0),
		Failed:  make(map[string]string),
		Skipped: make([]string, 0),
	}
}

// HasFailures returns true if any operations failed
func (r *OperationResult) HasFailures() bool {
	return len(r.Failed) > 0
}

// TotalProcessed returns the total number of servers processed
func (r *OperationResult) TotalProcessed() int {
	return len(r.Success) + len(r.Failed) + len(r.Skipped)
}

// LifecycleOutput holds the output for JSON mode
type LifecycleOutput struct {
	Status  string                 `json:"status"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// getServerNamesFromArgs resolves server names from args and --all flag
func getServerNamesFromArgs(ctx context.Context, args []string, all bool) ([]string, error) {
	if all && len(args) > 0 {
		return nil, fmt.Errorf("cannot specify server names with --all flag")
	}

	if all {
		// Get all servers
		servers, err := state.ListServers(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list servers: %w", err)
		}
		if len(servers) == 0 {
			return nil, fmt.Errorf("no servers found")
		}
		return servers, nil
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("no server names specified (use --all to target all servers)")
	}

	return args, nil
}

// loadServerForOperation loads a server state and validates it exists
func loadServerForOperation(ctx context.Context, name string) (*state.ServerState, error) {
	// Validate server name
	if err := state.ValidateServerName(name); err != nil {
		return nil, fmt.Errorf("invalid server name: %w", err)
	}

	// Load server state
	serverState, err := state.LoadServerState(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to load server state: %w", err)
	}

	// Validate container ID exists
	if serverState.ContainerID == "" {
		return nil, fmt.Errorf("server has no container (may need recreation)")
	}

	return serverState, nil
}

// outputOperationResult outputs the operation result in human or JSON format
func outputOperationResult(stdout io.Writer, operation string, result *OperationResult) error {
	jsonMode := isJSONMode()

	if jsonMode {
		return outputOperationJSON(stdout, operation, result)
	}

	return outputOperationHuman(stdout, operation, result)
}

// outputOperationHuman outputs results in human-readable format
func outputOperationHuman(stdout io.Writer, operation string, result *OperationResult) error {
	// Output successful operations
	for _, name := range result.Success {
		verb := getOperationVerb(operation, false)
		_, _ = fmt.Fprintf(stdout, "%s server '%s'\n", verb, name)
	}

	// Output skipped operations
	for _, name := range result.Skipped {
		reason := getSkipReason(operation)
		_, _ = fmt.Fprintf(stdout, "Server '%s' %s (skipped)\n", name, reason)
	}

	// Output failed operations
	for name, errMsg := range result.Failed {
		verb := getOperationVerb(operation, true)
		_, _ = fmt.Fprintf(stdout, "%s '%s': %s\n", verb, name, errMsg)
	}

	// Return error if any operations failed
	if result.HasFailures() {
		return fmt.Errorf("%d server(s) failed", len(result.Failed))
	}

	return nil
}

// outputOperationJSON outputs results in JSON format
func outputOperationJSON(stdout io.Writer, operation string, result *OperationResult) error {
	status := "success"
	if result.HasFailures() {
		if len(result.Success) > 0 {
			status = "partial"
		} else {
			status = "error"
		}
	}

	data := make(map[string]interface{})
	data[operation] = result.Success

	if len(result.Skipped) > 0 {
		data["skipped"] = result.Skipped
	}

	if len(result.Failed) > 0 {
		data["failed"] = result.Failed
	}

	output := LifecycleOutput{
		Status: status,
		Data:   data,
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// getOperationVerb returns the appropriate verb for the operation
func getOperationVerb(operation string, failed bool) string {
	if failed {
		switch operation {
		case "started":
			return "Failed to start"
		case "stopped":
			return "Failed to stop"
		case "restarted":
			return "Failed to restart"
		default:
			return "Failed to process"
		}
	}

	switch operation {
	case "started":
		return "Started"
	case "stopped":
		return "Stopped"
	case "restarted":
		return "Restarted"
	default:
		return "Processed"
	}
}

// getSkipReason returns the reason for skipping based on operation
func getSkipReason(operation string) string {
	switch operation {
	case "started":
		return "already running"
	case "stopped":
		return "already stopped"
	case "restarted":
		return "not running"
	default:
		return "skipped"
	}
}

// createContainerClient creates a new container client
func createContainerClient(ctx context.Context) (container.Client, error) {
	client, err := container.NewClient(ctx, container.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to container runtime: %w", err)
	}
	return client, nil
}

// outputLifecycleError outputs an error message
func outputLifecycleError(stdout io.Writer, jsonMode bool, err error) error {
	if jsonMode {
		output := LifecycleOutput{
			Status: "error",
			Error:  err.Error(),
		}
		_ = json.NewEncoder(stdout).Encode(output)
	}
	return err
}

// checkContainerExists verifies that a container exists and returns its info
func checkContainerExists(ctx context.Context, client container.Client, containerID string) (*container.ContainerInfo, error) {
	info, err := client.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found or inaccessible: %w", err)
	}
	return info, nil
}

// updateServerStatus updates server status and timestamps
func updateServerStatus(ctx context.Context, serverState *state.ServerState, status state.ServerStatus) error {
	serverState.Status = status

	// Update timestamps based on status
	switch status {
	case state.StatusRunning:
		serverState.LastStarted = serverState.UpdatedAt
	case state.StatusStopped:
		serverState.LastStopped = serverState.UpdatedAt
	}

	// Save updated state
	if err := state.SaveServerState(ctx, serverState); err != nil {
		slog.Warn("failed to save server state after status update",
			"server", serverState.Name,
			"error", err)
		return fmt.Errorf("failed to save server state: %w", err)
	}

	return nil
}
