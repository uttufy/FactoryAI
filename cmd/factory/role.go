package main

import (
	"github.com/spf13/cobra"
)

// getRoleCmds returns role management commands
func getRoleCmds() []*cobra.Command {
	return []*cobra.Command{
		roleListCmd,
		roleSetCmd,
		roleClearCmd,
	}
}

var roleListCmd = &cobra.Command{
	Use:   "role list",
	Short: "List available roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: List roles from config
		printInfo("Available roles:")
		println("  - developer")
		println("  - reviewer")
		println("  - architect")
		println("  - tester")
		return nil
	},
}

var roleSetCmd = &cobra.Command{
	Use:   "role set <role>",
	Short: "Set current operator role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		role := args[0]
		// TODO: Set role in config
		printSuccess("Role set to '%s'", role)
		return nil
	},
}

var roleClearCmd = &cobra.Command{
	Use:   "role clear",
	Short: "Clear current role",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Clear role from config
		printSuccess("Role cleared")
		return nil
	},
}
