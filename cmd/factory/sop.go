package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getSOPCmds returns SOP management commands
func getSOPCmds() []*cobra.Command {
	return []*cobra.Command{
		sopListCmd,
		sopShowCmd,
		sopExecuteCmd,
	}
}

var sopListCmd = &cobra.Command{
	Use:   "sop list",
	Short: "List SOPs",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: List SOPs from store
		printInfo("No SOPs found")
		return nil
	},
}

var sopShowCmd = &cobra.Command{
	Use:   "sop show <id>",
	Short: "Show SOP details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		// TODO: Show SOP details
		fmt.Printf("SOP: %s\n", id)
		return nil
	},
}

var sopExecuteCmd = &cobra.Command{
	Use:   "sop execute <id>",
	Short: "Execute an SOP",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		// TODO: Execute SOP
		printSuccess("SOP '%s' executed", id)
		return nil
	},
}
