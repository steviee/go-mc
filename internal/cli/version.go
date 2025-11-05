package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// VersionInfo contains version information for the application
type VersionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	BuiltBy string `json:"built_by"`
}

// NewVersionCommand creates the version command
func NewVersionCommand(version, commit, date, builtBy string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information including build commit and date.",
		Example: `  # Display version information
  go-mc version

  # Output in JSON format
  go-mc version --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printVersion(cmd.OutOrStdout(), version, commit, date, builtBy)
		},
	}

	return cmd
}

// printVersion prints version information in the appropriate format
func printVersion(w io.Writer, version, commit, date, builtBy string) error {
	info := VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	}

	if IsJSONOutput() {
		return printVersionJSON(w, info)
	}

	return printVersionText(w, info)
}

// printVersionJSON prints version information in JSON format
func printVersionJSON(w io.Writer, info VersionInfo) error {
	output := struct {
		Status string      `json:"status"`
		Data   VersionInfo `json:"data"`
	}{
		Status: "success",
		Data:   info,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}

	return nil
}

// printVersionText prints version information in human-readable format
func printVersionText(w io.Writer, info VersionInfo) error {
	if _, err := fmt.Fprintf(w, "go-mc version %s\n", info.Version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	if _, err := fmt.Fprintf(w, "Commit: %s\n", info.Commit); err != nil {
		return fmt.Errorf("write commit: %w", err)
	}
	if _, err := fmt.Fprintf(w, "Built: %s\n", info.Date); err != nil {
		return fmt.Errorf("write date: %w", err)
	}
	if _, err := fmt.Fprintf(w, "Built by: %s\n", info.BuiltBy); err != nil {
		return fmt.Errorf("write built by: %w", err)
	}

	return nil
}
