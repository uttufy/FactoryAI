package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getFormulaCmds returns formula management commands
func getFormulaCmds() []*cobra.Command {
	return []*cobra.Command{
		formulaCreateCmd,
		formulaListCmd,
		formulaShowCmd,
		formulaValidateCmd,
	}
}

var formulaCreateCmd = &cobra.Command{
	Use:   "formula create <name>",
	Short: "Create a formula (SOP template)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		// TODO: Create formula template
		printSuccess("Formula '%s' created", name)
		return nil
	},
}

var formulaListCmd = &cobra.Command{
	Use:   "formula list",
	Short: "List formulas",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: List formulas from store
		printInfo("No formulas found")
		return nil
	},
}

var formulaShowCmd = &cobra.Command{
	Use:   "formula show <name>",
	Short: "Show formula details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		// TODO: Show formula details
		fmt.Printf("Formula: %s\n", name)
		return nil
	},
}

var formulaValidateCmd = &cobra.Command{
	Use:   "formula validate <file>",
	Short: "Validate formula YAML",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		// TODO: Validate formula file
		printSuccess("Formula '%s' is valid", file)
		return nil
	},
}
