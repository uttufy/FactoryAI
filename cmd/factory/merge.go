package main

import (
	"context"
	"fmt"
	"time"

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

		if assemblyMgr == nil {
			return fmt.Errorf("assembly manager not initialized")
		}

		ctx := context.Background()
		queue := assemblyMgr.GetQueue(ctx)

		pending := assemblyMgr.GetPendingCount()
		conflicted := assemblyMgr.GetConflictedCount()

		fmt.Println("Merge Queue Status:")
		fmt.Printf("  Pending: %d\n", pending)
		fmt.Printf("  Conflicted: %d\n", conflicted)
		fmt.Printf("  Total in queue: %d\n", len(queue))

		if len(queue) > 0 {
			fmt.Println("\nQueue:")
			for _, mr := range queue {
				fmt.Printf("  %s: %s [%s]\n", mr.ID, mr.BeadID, mr.Status)
				if mr.Branch != "" {
					fmt.Printf("    Branch: %s\n", mr.Branch)
				}
				if len(mr.Conflicts) > 0 {
					fmt.Printf("    Conflicts: %v\n", mr.Conflicts)
				}
			}
		}
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

		if assemblyMgr == nil {
			return fmt.Errorf("assembly manager not initialized")
		}

		ctx := context.Background()
		queue := assemblyMgr.GetQueue(ctx)

		if len(queue) == 0 {
			printInfo("No pending merges")
			return nil
		}

		fmt.Println("Pending Merges:")
		for _, mr := range queue {
			age := time.Since(mr.SubmittedAt).Round(time.Second)
			fmt.Printf("  %s\n", mr.ID)
			fmt.Printf("    Bead: %s\n", mr.BeadID)
			fmt.Printf("    Station: %s\n", mr.StationID)
			fmt.Printf("    Branch: %s\n", mr.Branch)
			fmt.Printf("    Status: %s (submitted %s ago)\n", mr.Status, age)
			if mr.Error != "" {
				fmt.Printf("    Error: %s\n", mr.Error)
			}
		}
		return nil
	},
}

var mergeApproveCmd = &cobra.Command{
	Use:   "merge approve <mr-id>",
	Short: "Approve and execute a merge",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		if assemblyMgr == nil {
			return fmt.Errorf("assembly manager not initialized")
		}

		ctx := context.Background()
		mrID := args[0]

		// Check for conflicts first
		conflicts, err := assemblyMgr.CheckConflicts(ctx, mrID)
		if err != nil {
			return fmt.Errorf("checking conflicts: %w", err)
		}

		if len(conflicts) > 0 {
			printError("Merge has conflicts:")
			for _, f := range conflicts {
				fmt.Printf("  - %s\n", f)
			}
			return fmt.Errorf("resolve conflicts before approving")
		}

		// Execute the merge
		if err := assemblyMgr.Merge(ctx, mrID); err != nil {
			return fmt.Errorf("merge failed: %w", err)
		}

		printSuccess("Merge %s completed successfully", mrID)
		return nil
	},
}

var mergeBlockCmd = &cobra.Command{
	Use:   "merge block <mr-id> <reason>",
	Short: "Block a merge request",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		if assemblyMgr == nil {
			return fmt.Errorf("assembly manager not initialized")
		}

		ctx := context.Background()
		mrID := args[0]
		reason := args[1]

		if err := assemblyMgr.Escalate(ctx, mrID, reason); err != nil {
			return fmt.Errorf("blocking merge: %w", err)
		}

		printSuccess("Merge %s blocked: %s", mrID, reason)
		return nil
	},
}
