// Package cli implements the FactoryAI command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "none"
)

var rootCmd = &cobra.Command{
	Use:   "factory",
	Short: "FactoryAI - Multi-agent workspace manager",
	Long: `FactoryAI is a multi-agent workspace manager that orchestrates 
parallel AI agents working on software development tasks.

It uses manufacturing factory concepts:
- Stations: Isolated git worktrees where AI agents work
- Operators: AI agents (Claude) working at stations
- Travelers: Work orders that move through stations
- SOPs: Standard Operating Procedures (DAG workflows)`,
	Version: Version,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for testing
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

// exitWithError prints an error and exits with code 1
func exitWithError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
