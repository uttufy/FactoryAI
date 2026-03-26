package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/workflow"
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

		// Create formulas directory if it doesn't exist
		formulasDir := filepath.Join(projectPath, "formulas")
		if err := os.MkdirAll(formulasDir, 0755); err != nil {
			return fmt.Errorf("creating formulas directory: %w", err)
		}

		// Create a basic formula template
		template := fmt.Sprintf(`name = "%s"
description = "Description of %s formula"

[[steps]]
name = "step-1"
description = "First step description"
assignee = "developer"
dependencies = []
acceptance = "Acceptance criteria for step 1"

[[steps]]
name = "step-2"
description = "Second step description"
assignee = "reviewer"
dependencies = ["step-1"]
acceptance = "Acceptance criteria for step 2"
`, name, name)

		filePath := filepath.Join(formulasDir, name+".toml")
		if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
			return fmt.Errorf("writing formula file: %w", err)
		}

		printSuccess("Formula '%s' created at %s", name, filePath)
		return nil
	},
}

var formulaListCmd = &cobra.Command{
	Use:   "formula list",
	Short: "List formulas",
	RunE: func(cmd *cobra.Command, args []string) error {
		formulasDir := filepath.Join(projectPath, "formulas")

		files, err := filepath.Glob(filepath.Join(formulasDir, "*.toml"))
		if err != nil {
			return fmt.Errorf("listing formulas: %w", err)
		}

		if len(files) == 0 {
			printInfo("No formulas found in %s", formulasDir)
			return nil
		}

		fmt.Println("Formulas:")
		for _, f := range files {
			// Try to load and display basic info
			formula, err := workflow.LoadFormula(f)
			if err != nil {
				fmt.Printf("  %s (error loading: %v)\n", filepath.Base(f), err)
				continue
			}
			fmt.Printf("  %s: %s (%d steps)\n", filepath.Base(f), formula.Name, len(formula.Steps))
		}
		return nil
	},
}

var formulaShowCmd = &cobra.Command{
	Use:   "formula show <file>",
	Short: "Show formula details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		// If not an absolute path, look in formulas directory
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectPath, "formulas", file)
			if filepath.Ext(file) == "" {
				file = file + ".toml"
			}
		}

		formula, err := workflow.LoadFormula(file)
		if err != nil {
			return fmt.Errorf("loading formula: %w", err)
		}

		fmt.Printf("Formula: %s\n", formula.Name)
		if formula.Description != "" {
			fmt.Printf("Description: %s\n", formula.Description)
		}
		fmt.Printf("File: %s\n", file)
		fmt.Println("\nSteps:")
		for i, step := range formula.Steps {
			fmt.Printf("  %d. %s\n", i+1, step.Name)
			if step.Description != "" {
				fmt.Printf("     Description: %s\n", step.Description)
			}
			if step.Assignee != "" {
				fmt.Printf("     Assignee: %s\n", step.Assignee)
			}
			if len(step.Dependencies) > 0 {
				fmt.Printf("     Dependencies: %v\n", step.Dependencies)
			}
			if step.Acceptance != "" {
				fmt.Printf("     Acceptance: %s\n", step.Acceptance)
			}
		}
		return nil
	},
}

var formulaValidateCmd = &cobra.Command{
	Use:   "formula validate <file>",
	Short: "Validate formula TOML",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		// If not an absolute path, look in formulas directory
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectPath, "formulas", file)
			if filepath.Ext(file) == "" {
				file = file + ".toml"
			}
		}

		formula, err := workflow.LoadFormula(file)
		if err != nil {
			return fmt.Errorf("loading formula: %w", err)
		}

		// Validate the formula
		if err := formula.Validate(); err != nil {
			printError("Formula validation failed: %v", err)
			return err
		}

		printSuccess("Formula '%s' is valid (%d steps)", formula.Name, len(formula.Steps))
		return nil
	},
}
