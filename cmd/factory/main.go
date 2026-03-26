package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "factory",
	Short: "FactoryAI - Multi-agent workspace manager",
	Long:  "A manufacturing-plant-inspired multi-agent workspace manager in Go.",
}

func init() {
	// Factory management commands are at root level (init, boot, status, etc.)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(bootCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)

	// Add all parent commands (each with their subcommands)
	rootCmd.AddCommand(getStationCmd())
	rootCmd.AddCommand(getOperatorCmd())
	rootCmd.AddCommand(getWorkCellCmd())
	rootCmd.AddCommand(getJobCmd())
	rootCmd.AddCommand(getTravelerCmd())
	rootCmd.AddCommand(getBatchCmd())
	rootCmd.AddCommand(getFormulaCmd())
	rootCmd.AddCommand(getSOPCmd())
	rootCmd.AddCommand(getExecutionCmd())
	rootCmd.AddCommand(getSupportCmd())
	rootCmd.AddCommand(getMergeCmd())
	rootCmd.AddCommand(getMailCmd())
	rootCmd.AddCommand(getRoleCmd())

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./configs/factory.yaml", "Path to factory config")
	rootCmd.PersistentFlags().StringVar(&projectPath, "project-path", ".", "Project path")
}
