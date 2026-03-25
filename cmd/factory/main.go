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
	// Add all command groups
	rootCmd.AddCommand(getFactoryCmds()...)
	rootCmd.AddCommand(getStationCmds()...)
	rootCmd.AddCommand(getOperatorCmds()...)
	rootCmd.AddCommand(getWorkCellCmds()...)
	rootCmd.AddCommand(getJobCmds()...)
	rootCmd.AddCommand(getTravelerCmds()...)
	rootCmd.AddCommand(getBatchCmds()...)
	rootCmd.AddCommand(getFormulaCmds()...)
	rootCmd.AddCommand(getSOPCmds()...)
	rootCmd.AddCommand(getExecutionCmds()...)
	rootCmd.AddCommand(getSupportCmds()...)
	rootCmd.AddCommand(getMergeCmds()...)
	rootCmd.AddCommand(getMailCmds()...)
	rootCmd.AddCommand(getRoleCmds()...)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./configs/factory.yaml", "Path to factory config")
	rootCmd.PersistentFlags().StringVar(&projectPath, "project-path", ".", "Project path")
}
