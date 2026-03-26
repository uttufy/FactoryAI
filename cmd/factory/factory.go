package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/assembly"
	"github.com/uttufy/FactoryAI/internal/batch"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/director"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/mail"
	"github.com/uttufy/FactoryAI/internal/operator"
	"github.com/uttufy/FactoryAI/internal/planner"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/supervisor"
	"github.com/uttufy/FactoryAI/internal/support"
	"github.com/uttufy/FactoryAI/internal/tmux"
	"github.com/uttufy/FactoryAI/internal/traveler"
	"github.com/uttufy/FactoryAI/internal/workflow"
)

// getFactoryCmds returns factory management commands
func getFactoryCmds() []*cobra.Command {
	return []*cobra.Command{
		initCmd,
		statusCmd,
		bootCmd,
		shutdownCmd,
		pauseCmd,
		resumeCmd,
	}
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new factory",
	Long:  "Initialize the factory directory, database, and configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create factory directory
		if err := os.MkdirAll(factoryDir, 0755); err != nil {
			return fmt.Errorf("failed to create factory directory: %w", err)
		}

		// Initialize store
		var err error
		storeInstance, err = store.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		if err := storeInstance.Migrate(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		// Initialize tmux manager
		tmuxInstance, err = tmux.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize tmux: %w", err)
		}

		// Initialize beads client
		beadsClient, err = beads.NewClient("beads", projectPath)
		if err != nil {
			return fmt.Errorf("failed to initialize beads client: %w", err)
		}

		printSuccess("Factory initialized at %s", factoryDir)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show factory status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		ctx := context.Background()
		status, err := directorInstance.GetStatus(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("Factory Status: %s\n", map[bool]string{true: "Running", false: "Stopped"}[status.Running])
		fmt.Printf("Uptime: %s\n", status.Uptime)
		fmt.Printf("Active Jobs: %d\n", status.ActiveJobs)
		fmt.Printf("Pending Batches: %d\n", status.PendingBatches)
		fmt.Printf("Last Activity: %s\n", status.LastActivity.Format("2006-01-02 15:04:05"))
		fmt.Println("\nStations:")
		for _, s := range status.Stations {
			job := s.CurrentJob
			if job == "" {
				job = "idle"
			}
			fmt.Printf("  %s (%s): %s\n", s.Name, s.Status, job)
		}
		return nil
	},
}

var bootCmd = &cobra.Command{
	Use:   "boot",
	Short: "Start all stations",
	Long:  "Initialize and start the factory director, all stations, and support services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		// Initialize store if not already
		if storeInstance == nil {
			storeInstance, err = store.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to initialize store: %w", err)
			}
		}

		// Initialize event bus
		eventsInstance = events.NewEventBus(1000, storeInstance)

		// Initialize tmux
		if tmuxInstance == nil {
			tmuxInstance, err = tmux.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize tmux: %w", err)
			}
		}

		// Initialize beads client
		if beadsClient == nil {
			beadsClient, err = beads.NewClient("beads", projectPath)
			if err != nil {
				return fmt.Errorf("failed to initialize beads client: %w", err)
			}
		}

		// Initialize components
		stationManager = station.NewManager(projectPath, eventsInstance, storeInstance, tmuxInstance, maxStations)
		operatorPool = operator.NewPool(stationManager, eventsInstance, storeInstance, tmuxInstance, beadsClient)
		dagEngine = workflow.NewDAGEngine(eventsInstance)

		travelerMgr = traveler.NewManager(beadsClient, eventsInstance, storeInstance)
		plannerInstance = planner.NewPlanner(eventsInstance, storeInstance, travelerMgr, stationManager, beadsClient, nil)
		supportService = support.NewService(eventsInstance, storeInstance, tmuxInstance)
		supervisorInst = supervisor.NewSupervisor(eventsInstance, storeInstance, tmuxInstance)
		batchMgr = batch.NewManager(beadsClient, eventsInstance, storeInstance)

		// Initialize assembly (merge queue) and mail system
		assemblyMgr = assembly.NewAssembly(projectPath, eventsInstance, storeInstance, beadsClient)
		mailSystem = mail.NewService(beadsClient)

		// Initialize director
		directorInstance = director.NewDirector(
			plannerInstance,
			stationManager,
			supervisorInst,
			supportService,
			eventsInstance,
			storeInstance,
			tmuxInstance,
			beadsClient,
		)

		// Start director
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := directorInstance.Start(ctx); err != nil {
			return fmt.Errorf("failed to start director: %w", err)
		}

		printSuccess("Factory booted successfully")

		// Wait for shutdown signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		printInfo("Shutting down...")
		return shutdownDirector()
	},
}

var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Graceful shutdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}
		return shutdownDirector()
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause factory (Director only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}
		if err := directorInstance.Pause(); err != nil {
			return err
		}
		printSuccess("Factory paused")
		return nil
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume factory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}
		if err := directorInstance.Resume(); err != nil {
			return err
		}
		printSuccess("Factory resumed")
		return nil
	},
}

func init() {
	bootCmd.Flags().IntVarP(&maxStations, "max-stations", "m", 5, "Maximum number of stations")
}
