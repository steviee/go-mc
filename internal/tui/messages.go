package tui

import "time"

// tickMsg is sent on every auto-refresh tick
type tickMsg time.Time

// serversLoadedMsg is sent when server list is loaded
type serversLoadedMsg struct {
	servers []ServerInfo
	err     error
}

// serverActionMsg is sent when a server action completes
type serverActionMsg struct {
	action string // "start", "stop", "restart", "delete"
	server string
	err    error
}

// errorMsg is sent when an error occurs
type errorMsg struct {
	err error
}

// clearErrorMsg is sent to clear the error message
type clearErrorMsg struct{}
