// Package cli implements traveler management commands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// travelerCmd represents the traveler command
var travelerCmd = &cobra.Command{
	Use:   "traveler",
	Short: "Manage travelers (work orders)",
	Long:  `Manage travelers - work orders that move through stations.`,
}

// travelerAttachCmd represents the traveler attach command
var travelerAttachCmd = &cobra.Command{
	Use:   "attach <station-id> <job-id>",
	Short: "Attach work to a station",
	Long:  `Attach a job (bead) to a station's traveler.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		stationID := args[0]
		beadID := args[1]
		if err := attachTraveler(stationID, beadID); err != nil {
			exitWithError("Failed to attach traveler", err)
		}
	},
}

// travelerShowCmd represents the traveler show command
var travelerShowCmd = &cobra.Command{
	Use:   "show <station-id>",
	Short: "Show station's traveler",
	Long:  `Show the current traveler for a station.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stationID := args[0]
		if err := showTraveler(stationID); err != nil {
			exitWithError("Failed to show traveler", err)
		}
	},
}

// travelerClearCmd represents the traveler clear command
var travelerClearCmd = &cobra.Command{
	Use:   "clear <station-id>",
	Short: "Clear station's traveler",
	Long:  `Clear the traveler from a station.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stationID := args[0]
		if err := clearTraveler(stationID); err != nil {
			exitWithError("Failed to clear traveler", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(travelerCmd)
	travelerCmd.AddCommand(travelerAttachCmd)
	travelerCmd.AddCommand(travelerShowCmd)
	travelerCmd.AddCommand(travelerClearCmd)
}

func attachTraveler(stationID, beadID string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	if err := factory.Planner.Dispatch(context.Background(), beadID, stationID); err != nil {
		return err
	}

	fmt.Printf("Traveler attached successfully!\n")
	fmt.Printf("  Station: %s\n", stationID)
	fmt.Printf("  Bead: %s\n", beadID)

	return nil
}

func showTraveler(stationID string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	traveler, err := factory.BeadsClient.GetTraveler(stationID)
	if err != nil {
		return fmt.Errorf("getting traveler: %w", err)
	}

	fmt.Println("TRAVELER DETAILS")
	fmt.Println("================")
	fmt.Printf("  ID: %s\n", traveler.ID)
	fmt.Printf("  Station: %s\n", traveler.StationID)
	fmt.Printf("  Bead ID: %s\n", traveler.BeadID)
	fmt.Printf("  Status: %s\n", traveler.Status)
	fmt.Printf("  Priority: %d\n", traveler.Priority)

	if traveler.SOPID != "" {
		fmt.Printf("  SOP ID: %s\n", traveler.SOPID)
	}
	if traveler.ReworkCount > 0 {
		fmt.Printf("  Rework Count: %d\n", traveler.ReworkCount)
		fmt.Printf("  Rework Reason: %s\n", traveler.ReworkReason)
	}

	fmt.Printf("  Attached: %s\n", traveler.AttachedAt.Format("2006-01-02 15:04:05"))

	if traveler.StartedAt != nil {
		fmt.Printf("  Started: %s\n", traveler.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if traveler.CompletedAt != nil {
		fmt.Printf("  Completed: %s\n", traveler.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if traveler.Result != "" {
		fmt.Printf("  Result: %s\n", traveler.Result)
	}
	if traveler.Error != "" {
		fmt.Printf("  Error: %s\n", traveler.Error)
	}

	return nil
}

func clearTraveler(stationID string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	if err := factory.BeadsClient.ClearTraveler(stationID); err != nil {
		return fmt.Errorf("clearing traveler: %w", err)
	}

	fmt.Printf("Traveler cleared from station '%s'.\n", stationID)
	return nil
}
