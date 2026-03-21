package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/factory"
	"github.com/uttufy/FactoryAI/internal/job"
	"github.com/uttufy/FactoryAI/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "factory",
	Short: "FactoryAI - Multi-agent workspace manager",
	Long:  "A manufacturing-plant-inspired multi-agent workspace manager in Go.",
}

var (
	blueprintPath string
	task          string
	noTUI         bool
)

var runCmd = &cobra.Command{
	Use:   "run --blueprint <path> --task <task>",
	Short: "Run a factory blueprint",
	Long:  "Execute a factory blueprint with the given task.",
	Args:  cobra.NoArgs,
	RunE:  runFactory,
}

var blueprintsDir string

var listBlueprintsCmd = &cobra.Command{
	Use:   "list-blueprints [--dir <path>]",
	Short: "List available blueprints",
	Long:  "List all available blueprint YAML files in the specified directory.",
	RunE:  listBlueprints,
}

func init() {
	runCmd.Flags().StringVarP(&blueprintPath, "blueprint", "b", "", "Path to blueprint YAML")
	runCmd.Flags().StringVarP(&task, "task", "t", "", "Task to execute")
	runCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Disable TUI, print progress to stdout")
	runCmd.MarkFlagRequired("blueprint")
	runCmd.MarkFlagRequired("task")

	listBlueprintsCmd.Flags().StringVarP(&blueprintsDir, "dir", "d", "./blueprints", "Directory containing blueprints")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listBlueprintsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runFactory(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	claudeBin := os.Getenv("CLAUDE_BIN")

	blueprint, err := config.LoadBlueprint(blueprintPath)
	if err != nil {
		return fmt.Errorf("failed to load blueprint: %w", err)
	}

	agent, err := agents.NewAgent("claude", claudeBin)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	f, err := factory.New(blueprint, agent)
	if err != nil {
		return fmt.Errorf("failed to create factory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	eventsChan := make(chan events.Event, 100)

	if noTUI {
		return runWithoutTUI(ctx, f, blueprint, task, eventsChan)
	}

	return runWithTUI(ctx, f, blueprint, task, eventsChan)
}

func runWithoutTUI(ctx context.Context, f *factory.Factory, blueprint *config.Blueprint, task string, eventsChan chan events.Event) error {
	fmt.Printf("🏭 Starting factory: %s\n", blueprint.Factory.Name)
	fmt.Println(strings.Repeat("─", 50))

	resultChan := make(chan *job.JobResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := f.Run(ctx, task, eventsChan)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	for {
		select {
		case evt := <-eventsChan:
			printEvent(evt)
			if evt.Type == events.EvtDone {
				fmt.Println("\n" + strings.Repeat("=", 50))
				fmt.Println("Final Output:")
				fmt.Println(strings.Repeat("=", 50))
				fmt.Println(evt.Output)
				return nil
			}
		case result := <-resultChan:
			fmt.Println("\n" + strings.Repeat("=", 50))
			fmt.Println("Final Output:")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Println(result.FinalOutput)
			return nil
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func printEvent(evt events.Event) {
	switch evt.Type {
	case events.EvtStationStarted:
		fmt.Printf("[START]    %s / %s\n", evt.LineName, evt.StationName)
	case events.EvtStationInspecting:
		fmt.Printf("[INSPECT]  %s / %s\n", evt.LineName, evt.StationName)
	case events.EvtStationDone:
		retryStr := ""
		if evt.Retries > 0 {
			retryStr = fmt.Sprintf(" (x%d)", evt.Retries+1)
		}
		fmt.Printf("[DONE]     %s / %s%s (%.1fs)\n",
			evt.LineName, evt.StationName, retryStr, evt.Duration.Seconds())
	case events.EvtStationFailed:
		fmt.Printf("[FAIL]     %s / %s: %v\n", evt.LineName, evt.StationName, evt.Error)
	case events.EvtMerging:
		fmt.Println("[MERGING]  Combining outputs...")
	}
}

func runWithTUI(ctx context.Context, f *factory.Factory, blueprint *config.Blueprint, task string, eventsChan chan events.Event) error {
	go func() {
		_, _ = f.Run(ctx, task, eventsChan)
	}()

	model := tui.NewModel(blueprint, eventsChan)
	p := tea.NewProgram(model)

	_, err := p.Run()
	return err
}

func listBlueprints(cmd *cobra.Command, args []string) error {
	entries, err := os.ReadDir(blueprintsDir)
	if err != nil {
		return fmt.Errorf("failed to read blueprints directory: %w", err)
	}

	fmt.Println("Available Blueprints:")
	fmt.Println(strings.Repeat("=", 50))

	found := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
			path := filepath.Join(blueprintsDir, entry.Name())
			bp, err := config.LoadBlueprint(path)
			if err != nil {
				fmt.Printf("\n%s: (error loading: %v)\n", entry.Name(), err)
				continue
			}

			found = true
			fmt.Printf("\n📄 %s\n", entry.Name())
			fmt.Printf("   Name: %s\n", bp.Factory.Name)
			fmt.Printf("   Description: %s\n", bp.Factory.Description)
			fmt.Printf("   Assembly Lines: %d\n", len(bp.Factory.AssemblyLines))
			for _, line := range bp.Factory.AssemblyLines {
				fmt.Printf("     - %s (%d stations)\n", line.Name, len(line.Stations))
			}
		}
	}

	if !found {
		fmt.Println("\nNo blueprint files found in", blueprintsDir)
	}

	return nil
}
