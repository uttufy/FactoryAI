// Package cli implements formula management commands.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// formulaCmd represents the formula command
var formulaCmd = &cobra.Command{
	Use:   "formula",
	Short: "Manage formulas (workflow recipes)",
	Long:  `Manage formulas - TOML-based workflow recipes that can be cooked into SOPs.`,
}

// formulaListCmd represents the formula list command
var formulaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available formulas",
	Long:  `List all available formulas in the formulas directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listFormulas(); err != nil {
			exitWithError("Failed to list formulas", err)
		}
	},
}

// formulaShowCmd represents the formula show command
var formulaShowCmd = &cobra.Command{
	Use:   "show <path>",
	Short: "Show formula details",
	Long:  `Show the contents and steps of a formula file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		if err := showFormula(path); err != nil {
			exitWithError("Failed to show formula", err)
		}
	},
}

// formulaCookCmd represents the formula cook command
var formulaCookCmd = &cobra.Command{
	Use:   "cook <path>",
	Short: "Cook a formula into a SOP",
	Long: `Cook a formula (TOML recipe) into a Standard Operating Procedure (SOP).

The resulting SOP can be instantiated and attached to stations.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		if err := cookFormula(path); err != nil {
			exitWithError("Failed to cook formula", err)
		}
	},
}

// runCmd represents the run command for executing formulas
var runCmd = &cobra.Command{
	Use:   "run --formula <path> --task <description>",
	Short: "Run a formula with a task",
	Long: `Run a formula with a specific task description.

This will cook the formula, create a SOP, and dispatch it to an available station.

Example:
  factory run --formula ./formulas/feature.toml --task "Build user authentication"`,
	Run: func(cmd *cobra.Command, args []string) {
		formulaPath, _ := cmd.Flags().GetString("formula")
		task, _ := cmd.Flags().GetString("task")

		if formulaPath == "" {
			exitWithError("Formula path required", fmt.Errorf("use --formula flag"))
		}
		if task == "" {
			exitWithError("Task description required", fmt.Errorf("use --task flag"))
		}

		if err := runFormula(formulaPath, task); err != nil {
			exitWithError("Failed to run formula", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(formulaCmd)
	formulaCmd.AddCommand(formulaListCmd)
	formulaCmd.AddCommand(formulaShowCmd)
	formulaCmd.AddCommand(formulaCookCmd)

	// Add run command
	runCmd.Flags().String("formula", "", "Path to formula file")
	runCmd.Flags().String("task", "", "Task description")
	rootCmd.AddCommand(runCmd)
}

func listFormulas() error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	formulas, err := factory.WorkflowEngine.ListFormulas()
	if err != nil {
		return fmt.Errorf("listing formulas: %w", err)
	}

	if len(formulas) == 0 {
		fmt.Println("No formulas found.")
		fmt.Println("\nFormulas are TOML files in the 'formulas/' directory.")
		fmt.Println("Create a formula file and run: factory formula cook <path>")
		return nil
	}

	fmt.Println("FORMULAS")
	fmt.Println("========")
	fmt.Printf("%-30s %-20s %s\n", "NAME", "STEPS", "PATH")
	fmt.Println("------------------------------------------------------------------------")

	for _, f := range formulas {
		fmt.Printf("%-30s %-20d %s\n",
			f.Name,
			len(f.Steps),
			f.Path,
		)
	}

	return nil
}

func showFormula(path string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	formula, err := factory.WorkflowEngine.LoadFormula(path)
	if err != nil {
		return fmt.Errorf("loading formula: %w", err)
	}

	fmt.Println("FORMULA DETAILS")
	fmt.Println("===============")
	fmt.Printf("  Name: %s\n", formula.Name)

	if formula.Description != "" {
		fmt.Printf("  Description: %s\n", formula.Description)
	}

	if len(formula.Variables) > 0 {
		fmt.Println("\n  Variables:")
		for k, v := range formula.Variables {
			fmt.Printf("    %s = %s\n", k, v)
		}
	}

	fmt.Println("\n  Steps:")
	for i, step := range formula.Steps {
		fmt.Printf("\n    %d. %s\n", i+1, step.Name)
		if step.Description != "" {
			fmt.Printf("       Description: %s\n", step.Description)
		}
		if step.Assignee != "" {
			fmt.Printf("       Assignee: %s\n", step.Assignee)
		}
		if len(step.Dependencies) > 0 {
			fmt.Printf("       Dependencies: %v\n", step.Dependencies)
		}
		if step.Acceptance != "" {
			fmt.Printf("       Acceptance: %s\n", step.Acceptance)
		}
	}

	return nil
}

func cookFormula(path string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	sop, err := factory.WorkflowEngine.CookFormula(path, nil)
	if err != nil {
		return fmt.Errorf("cooking formula: %w", err)
	}

	fmt.Printf("Formula cooked successfully!\n")
	fmt.Printf("  SOP ID: %s\n", sop.ID)
	fmt.Printf("  Name: %s\n", sop.Name)
	fmt.Printf("  Steps: %d\n", len(sop.Steps))

	return nil
}

func runFormula(path, task string) error {
	factory := GetFactory()
	if factory == nil {
		return fmt.Errorf("factory not running. Run 'factory boot' first")
	}

	// Cook the formula
	sop, err := factory.WorkflowEngine.CookFormula(path, map[string]string{
		"task": task,
	})
	if err != nil {
		return fmt.Errorf("cooking formula: %w", err)
	}

	fmt.Printf("Formula cooked into SOP: %s\n", sop.ID)

	// Create a bead for this task
	bead, err := factory.BeadsClient.Create("task", task)
	if err != nil {
		return fmt.Errorf("creating bead: %w", err)
	}

	fmt.Printf("Created task bead: %s\n", bead.ID)

	// Find an available station
	ctx := context.Background()
	stations := factory.StationManager.GetAvailable(ctx)
	if len(stations) == 0 {
		return fmt.Errorf("no available stations. Create one with 'factory station add <name>'")
	}

	// Dispatch to the first available station
	station := stations[0]
	if err := factory.Planner.Dispatch(nil, bead.ID, station.ID); err != nil {
		return fmt.Errorf("dispatching task: %w", err)
	}

	fmt.Printf("Task dispatched to station: %s\n", station.ID)
	fmt.Println("\nRun 'factory status' to monitor progress.")

	return nil
}
