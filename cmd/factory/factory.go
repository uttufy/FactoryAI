package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		// Get store instance
		st, err := getStore()
		if err != nil {
			return err
		}

		// Read status from database
		status, err := st.GetFactoryStatus()
		if err != nil {
			return fmt.Errorf("failed to get factory status: %w", err)
		}

		// Display status
		fmt.Printf("Factory Status: %s\n", status.BootStatus)
		if status.PID > 0 {
			fmt.Printf("PID: %d\n", status.PID)
		}
		if !status.StartedAt.IsZero() {
			fmt.Printf("Uptime: %s\n", time.Since(status.StartedAt).Round(time.Second))
		}

		// If factory is running and we have director instance, show more details
		if status.Running && directorInstance != nil {
			ctx := context.Background()
			directorStatus, err := directorInstance.GetStatus(ctx)
			if err == nil {
				fmt.Printf("Active Jobs: %d\n", directorStatus.ActiveJobs)
				fmt.Printf("Pending Batches: %d\n", directorStatus.PendingBatches)
				fmt.Printf("Last Activity: %s\n", directorStatus.LastActivity.Format("2006-01-02 15:04:05"))
				fmt.Println("\nStations:")
				for _, s := range directorStatus.Stations {
					job := s.CurrentJob
					if job == "" {
						job = "idle"
					}
					fmt.Printf("  %s (%s): %s\n", s.Name, s.Status, job)
				}
			}
		}

		return nil
	},
}

var bootCmd = &cobra.Command{
	Use:   "boot",
	Short: "Start all stations",
	Long: `Initialize and start the factory director, all stations, and support services.

This command runs as a persistent service. Use Ctrl+C to stop it, or run it
in the background with 'factory boot &' and use 'factory shutdown' to stop.

Examples:
  # Run in foreground (Ctrl+C to stop)
  factory boot

  # Run in background (recommended)
  factory boot &
  sleep 2  # Wait for initialization
  factory status

  # Stop background factory
  factory shutdown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		// Initialize store if not already
		if storeInstance == nil {
			storeInstance, err = store.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to initialize store: %w", err)
			}
		}

		// Check if already running (via database state)
		running, err := storeInstance.IsFactoryRunning()
		if err != nil {
			printInfo("Warning: could not check factory status: %v", err)
		}
		if running {
			return fmt.Errorf("factory already running. Use 'factory shutdown' first")
		}

		// Mark boot in progress in database
		pid := os.Getpid()
		if err := storeInstance.SetFactoryBooting(pid); err != nil {
			printInfo("Warning: could not set boot status: %v", err)
		}

		// Also mark legacy boot file for backwards compatibility
		if err := markFactoryBooted(); err != nil {
			printInfo("Warning: could not create boot status file: %v", err)
		}

		// Initialize event bus
		eventsInstance = events.NewEventBus(1000, storeInstance)

		// Initialize tmux
		if tmuxInstance == nil {
			tmuxInstance, err = tmux.NewManager()
			if err != nil {
				storeInstance.SetFactoryStopped()
				markFactoryShutdown()
				return fmt.Errorf("failed to initialize tmux: %w", err)
			}
		}

		// Initialize beads client
		if beadsClient == nil {
			beadsClient, err = beads.NewClient("beads", projectPath)
			if err != nil {
				storeInstance.SetFactoryStopped()
				markFactoryShutdown()
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
			storeInstance.SetFactoryStopped()
			markFactoryShutdown()
			return fmt.Errorf("failed to start director: %w", err)
		}

		// Mark as running in database
		if err := storeInstance.SetFactoryRunning(pid); err != nil {
			printInfo("Warning: failed to persist running state: %v", err)
		}

		printSuccess("Factory booted successfully")
		printInfo("Running in foreground. Press Ctrl+C to shutdown.")

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
	Long: `Gracefully shutdown the factory.

Works even if factory was started in background or crashed during boot.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get store instance
		st, err := getStore()
		if err != nil {
			return err
		}

		// Get current factory status
		status, err := st.GetFactoryStatus()
		if err != nil {
			return fmt.Errorf("failed to get factory status: %w", err)
		}

		// Clean up legacy boot status file
		markFactoryShutdown()

		// If factory is running in another process, send signal to that process
		if status.Running && status.PID > 0 {
			// Find the process and send SIGTERM
			process, err := os.FindProcess(int(status.PID))
			if err != nil {
				printInfo("Could not find factory process (PID %d), cleaning up state", status.PID)
				st.SetFactoryStopped()
				return nil
			}

			// Send SIGTERM to gracefully shutdown
			if err := process.Signal(syscall.SIGTERM); err != nil {
				// Process doesn't exist, clean up stale state
				printInfo("Factory process (PID %d) not running, cleaning up state", status.PID)
				st.SetFactoryStopped()
				return nil
			}

			printSuccess("Sent shutdown signal to factory (PID %d)", status.PID)
			return nil
		}

		// Check if director is running in this process (foreground case)
		if directorInstance != nil {
			return shutdownDirector()
		}

		printInfo("Factory not running (cleaned up status file)")
		st.SetFactoryStopped()
		return nil
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
