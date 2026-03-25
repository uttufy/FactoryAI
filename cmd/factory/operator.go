package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getOperatorCmds returns operator management commands
func getOperatorCmds() []*cobra.Command {
	return []*cobra.Command{
		operatorSpawnCmd,
		operatorListCmd,
		operatorStatusCmd,
		operatorDecommissionCmd,
	}
}

var operatorSpawnCmd = &cobra.Command{
	Use:   "operator spawn <station>",
	Short: "Spawn an operator at a station",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stationID := args[0]
		op, err := operatorPool.Spawn(ctx, stationID)
		if err != nil {
			return err
		}

		printSuccess("Operator '%s' spawned at station %s", op.ID, stationID)
		return nil
	},
}

var operatorListCmd = &cobra.Command{
	Use:   "operator list",
	Short: "List all operators",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		operators := operatorPool.List(ctx)
		if len(operators) == 0 {
			printInfo("No operators found")
			return nil
		}

		fmt.Println("Operators:")
		for _, op := range operators {
			task := op.CurrentTask
			if task == "" {
				task = "idle"
			}
			fmt.Printf("  %s (%s): %s - %s\n", op.ID, op.Name, op.Status, task)
		}
		return nil
	},
}

var operatorStatusCmd = &cobra.Command{
	Use:   "operator status <id>",
	Short: "Show operator status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]
		op, err := operatorPool.Get(ctx, id)
		if err != nil {
			return err
		}

		fmt.Printf("Operator: %s\n", op.Name)
		fmt.Printf("ID: %s\n", op.ID)
		fmt.Printf("Station: %s\n", op.StationID)
		fmt.Printf("Status: %s\n", op.Status)
		if op.CurrentTask != "" {
			fmt.Printf("Current Task: %s\n", op.CurrentTask)
		}
		fmt.Printf("Started: %s\n", op.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Last Heartbeat: %s\n", op.LastHeartbeat.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var operatorDecommissionCmd = &cobra.Command{
	Use:   "operator decommission <id>",
	Short: "Decommission an operator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		id := args[0]
		if err := operatorPool.Decommission(ctx, id); err != nil {
			return err
		}

		printSuccess("Operator '%s' decommissioned", id)
		return nil
	},
}
