package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// getTravelerCmds returns traveler management commands
func getTravelerCmds() []*cobra.Command {
	return []*cobra.Command{
		travelerAttachCmd,
		travelerShowCmd,
		travelerClearCmd,
	}
}

var travelerAttachCmd = &cobra.Command{
	Use:   "traveler attach <station> <job>",
	Short: "Attach work to station",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stationID := args[0]
		beadID := args[1]
		if err := travelerMgr.Attach(ctx, stationID, beadID); err != nil {
			return err
		}

		printSuccess("Work '%s' attached to station %s", beadID, stationID)
		return nil
	},
}

var travelerShowCmd = &cobra.Command{
	Use:   "traveler show <station>",
	Short: "Show station's traveler",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stationID := args[0]
		t, err := travelerMgr.GetTraveler(ctx, stationID)
		if err != nil {
			return err
		}

		fmt.Printf("Traveler ID: %s\n", t.ID)
		fmt.Printf("Station: %s\n", t.StationID)
		fmt.Printf("Bead: %s\n", t.BeadID)
		fmt.Printf("Status: %s\n", t.Status)
		fmt.Printf("Priority: %d\n", t.Priority)
		return nil
	},
}

var travelerClearCmd = &cobra.Command{
	Use:   "traveler clear <station>",
	Short: "Clear station's traveler",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		stationID := args[0]
		if err := travelerMgr.Detach(ctx, stationID); err != nil {
			return err
		}

		printSuccess("Traveler cleared from station %s", stationID)
		return nil
	},
}
