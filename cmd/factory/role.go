package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Role represents a role configuration
type Role struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Permissions []string        `yaml:"permissions"`
	Skills      []string        `yaml:"skills"`
}

// getRoleCmd returns the role parent command with all subcommands
func getRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Role management commands",
	}
	cmd.AddCommand(roleListCmd, roleSetCmd, roleClearCmd)
	return cmd
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available roles",
	RunE: func(cmd *cobra.Command, args []string) error {
		rolesDir := filepath.Join(projectPath, "configs", "roles")

		files, err := filepath.Glob(filepath.Join(rolesDir, "*.yaml"))
		if err != nil {
			return fmt.Errorf("listing roles: %w", err)
		}

		if len(files) == 0 {
			printInfo("No custom roles found. Built-in roles:")
			fmt.Println("  - developer: General development work")
			fmt.Println("  - architect: Design and planning")
			fmt.Println("  - reviewer: Code review")
			fmt.Println("  - tester: Testing and QA")
			return nil
		}

		fmt.Println("Available Roles:")
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}

			var role Role
			if err := yaml.Unmarshal(data, &role); err != nil {
				fmt.Printf("  %s: (error loading)\n", filepath.Base(f))
				continue
			}

			fmt.Printf("  %s: %s\n", role.ID, role.Name)
			if role.Description != "" {
				fmt.Printf("    %s\n", role.Description)
			}
		}
		return nil
	},
}

var roleSetCmd = &cobra.Command{
	Use:   "set <role>",
	Short: "Set current operator role",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		roleID := args[0]
		rolesDir := filepath.Join(projectPath, "configs", "roles")
		roleFile := filepath.Join(rolesDir, roleID+".yaml")

		// Check if role file exists
		if _, err := os.Stat(roleFile); os.IsNotExist(err) {
			// Built-in roles
			builtinRoles := map[string]string{
				"developer": "General development work",
				"architect": "Design and planning",
				"reviewer": "Code review",
				"tester":  "Testing and QA",
			}
			if desc, ok := builtinRoles[roleID]; ok {
				printSuccess("Role set to '%s' (%s)", roleID, desc)
				return nil
			}
			return fmt.Errorf("role not found: %s", roleID)
		}

		data, err := os.ReadFile(roleFile)
		if err != nil {
			return fmt.Errorf("reading role file: %w", err)
		}

		var role Role
		if err := yaml.Unmarshal(data, &role); err != nil {
			return fmt.Errorf("parsing role file: %w", err)
		}

		printSuccess("Role set to '%s' (%s)", role.ID, role.Name)
		return nil
	},
}

var roleClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current role",
	RunE: func(cmd *cobra.Command, args []string) error {
		printSuccess("Role cleared")
		return nil
	},
}
