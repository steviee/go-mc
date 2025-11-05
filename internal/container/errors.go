package container

import (
	"errors"
	"fmt"
)

// Sentinel errors for container operations.
var (
	// ErrDaemonNotRunning is returned when the container daemon is not running.
	ErrDaemonNotRunning = errors.New("container daemon is not running")

	// ErrSocketNotFound is returned when the container socket cannot be found.
	ErrSocketNotFound = errors.New("container socket not found")

	// ErrAPIVersionMismatch is returned when the API version is not supported.
	ErrAPIVersionMismatch = errors.New("API version not supported")

	// ErrPermissionDenied is returned when access to the socket is denied.
	ErrPermissionDenied = errors.New("permission denied accessing socket")

	// ErrNoRuntimeAvailable is returned when no container runtime is available.
	ErrNoRuntimeAvailable = errors.New("no container runtime available")
)

// RuntimeError wraps errors with runtime context.
type RuntimeError struct {
	Runtime string
	Socket  string
	Err     error
}

// Error returns the error message.
func (e *RuntimeError) Error() string {
	return fmt.Sprintf("%s: %v (socket: %s)", e.Runtime, e.Err, e.Socket)
}

// Unwrap returns the wrapped error.
func (e *RuntimeError) Unwrap() error {
	return e.Err
}

// NewRuntimeError creates a new RuntimeError.
func NewRuntimeError(runtime, socket string, err error) *RuntimeError {
	return &RuntimeError{
		Runtime: runtime,
		Socket:  socket,
		Err:     err,
	}
}
