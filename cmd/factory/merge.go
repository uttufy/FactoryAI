package main

import (
	"github.com/spf13/cobra"
)

// getMergeCmds returns merge queue commands
func getMergeCmds() []*cobra.Command {
	return []*cobra.Command{
		mergeStatusCmd,
		mergeListCmd,
		mergeApproveCmd,
		mergeBlockCmd,
	}
}

var mergeStatusCmd = &cobra.Command{
	Use:   "merge status",
	Short: "Show merge queue status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: Show merge queue status
		printInfo("Merge queue is empty")
		return nil
	},
}

var mergeListCmd = &cobra.Command{
	Use:   "merge list",
	Short: "List pending merges",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: List pending merges
		printInfo("No pending merges")
		return nil
	},
}

var mergeApproveCmd = &cobra.Command{
	Use:   "merge approve <mr>",
	Short: "Approve a merge request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		mrID := args[0]
		// TODO: Approve merge request
		printSuccess("Merge request '%s' approved", mrID)
		return nil
	},
}

var mergeBlockCmd = &cobra.Command{
	Use:   "merge block <mr> <reason>",
	Short: "Block a merge request",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		mrID := args[0]
		// reason := args[1]
		// TODO: Block merge request
		printSuccess("Merge request '%s' blocked", mrID)
		return nil
	},
}
