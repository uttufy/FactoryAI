// Package cli implements factory management commands.
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/director"
	"github.com/uttufy/FactoryAI/internal/events"
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

var (
	projectPath string
	cfgFile     string
)

// Factory represents the running factory
type Factory struct {
	Config         *config.Config
	Store          *store.Store
	EventBus       *events.EventBus
	TmuxManager    *tmux.Manager
	BeadsClient    *beads.Client
	StationManager *station.Manager
	OperatorPool   *operator.Pool
	Planner        *planner.Planner
	Supervisor     *supervisor.Supervisor
	SupportService *support.Service
	Director       *director.Director
	WorkflowEngine *workflow.DAGEngine
}

var factoryInstance *Factory

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new factory",
	Long: `Initialize a new FactoryAI workspace in the current directory.

This creates:
- .factory/ directory for state storage
- factory.yaml configuration file
- Initializes the SQLite database`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := initFactory(); err != nil {
			exitWithError("Failed to initialize factory", err)
		}
	},
}

// bootCmd represents the boot command
var bootCmd = &cobra.Command{
	Use:   "boot",
	Short: "Start the factory",
	Long: `Start the factory and all its services.

This initializes:
- Event bus
- Database connection
- Station manager
- Operator pool
- Support services`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := bootFactory(); err != nil {
			exitWithError("Failed to boot factory", err)
		}
	},
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show factory status",
	Long:  `Display the current status of the factory, stations, and operators.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showStatus(); err != nil {
			exitWithError("Failed to get status", err)
		}
	},
}

// shutdownCmd represents the shutdown command
var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Gracefully shutdown the factory",
	Long: `Gracefully shutdown the factory.

This will:
- Complete in-progress work
- Release all leases
- Close database connections`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := shutdownFactory(); err != nil {
			exitWithError("Failed to shutdown factory", err)
		}
	},
}

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause the factory",
	Long:  `Pause the factory (stop dispatching new work).`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := pauseFactory(); err != nil {
			exitWithError("Failed to pause factory", err)
		}
	},
}

// resumeCmd represents the resume command
var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the factory",
	Long:  `Resume the factory (continue dispatching work).`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := resumeFactory(); err != nil {
			exitWithError("Failed to resume factory", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(bootCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&projectPath, "path", "p", ".", "Project path")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file path")
}

func initFactory() error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	fmt.Printf("Initializing factory in %s\n", absPath)

	// Create factory directory
	if err := config.EnsureFactoryDir(absPath); err != nil {
		return err
	}

	// Create default config
	cfg := config.DefaultConfig()
	cfg.Factory.ProjectPath = absPath

	configPath := filepath.Join(absPath, "factory.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := cfg.Save(configPath); err != nil {
			return err
		}
		fmt.Printf("Created config file: %s\n", configPath)
	} else {
		fmt.Printf("Config file already exists: %s\n", configPath)
	}

	// Initialize database
	dbPath := filepath.Join(absPath, cfg.Database.Path)
	s, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	fmt.Printf("Created database: %s\n", dbPath)

	// Initialize beads
	fmt.Println("\nInitializing beads...")
	if err := initializeBeads(cfg, absPath); err != nil {
		fmt.Printf("Warning: beads initialization failed: %v\n", err)
		fmt.Println("You may need to run 'bd init' manually.")
	}

	fmt.Println("\nFactory initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit factory.yaml to configure your factory")
	fmt.Println("  2. Run 'factory boot' to start the factory")
	fmt.Println("  3. Run 'factory station add <name>' to add stations")

	return nil
}

// runBeadsDoctor runs beads doctor and returns error if critical issues found
func runBeadsDoctor(cfg *config.Config, absPath string) error {
	// Create beads client
	beadsClient, err := beads.NewClient(cfg.Beads.BinaryPath, absPath)
	if err != nil {
		return fmt.Errorf("creating beads client: %w", err)
	}

	if err := beadsClient.Doctor(); err != nil {
		return err
	}

	fmt.Println("Beads doctor check passed.")
	return nil
}

// initializeBeads initializes beads and installs git hooks
func initializeBeads(cfg *config.Config, absPath string) error {
	// Create beads client
	beadsClient, err := beads.NewClient(cfg.Beads.BinaryPath, absPath)
	if err != nil {
		return fmt.Errorf("creating beads client: %w", err)
	}

	// Check if beads is already initialized
	if beadsClient.IsInitialized() {
		fmt.Println("Beads already initialized.")
	} else {
		// Get prefix from directory name
		prefix := filepath.Base(absPath)
		fmt.Printf("Running 'bd init --prefix %s'...\n", prefix)
		if err := beadsClient.Init(prefix); err != nil {
			return err
		}
		fmt.Println("Beads initialized successfully.")
	}

	// Install git hooks
	fmt.Println("Installing git hooks...")
	if err := beadsClient.InstallHooks(); err != nil {
		fmt.Printf("Warning: could not install git hooks: %v\n", err)
	} else {
		fmt.Println("Git hooks installed successfully.")
	}

	return nil
}

func bootFactory() error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	// Load config
	cfg, err := loadConfig(absPath)
	if err != nil {
		return err
	}

	fmt.Printf("Booting factory in %s\n", absPath)

	// Run beads doctor to check system health
	fmt.Println("Running beads doctor...")
	if err := runBeadsDoctor(cfg, absPath); err != nil {
		return fmt.Errorf("beads doctor check failed:\n%w\n\nPlease fix beads issues before booting the factory.\nRun 'bd doctor' for details.", err)
	}

	// Initialize components
	factory, err := initializeFactory(cfg, absPath)
	if err != nil {
		return err
	}

	factoryInstance = factory

	// Start the factory
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal...")
		cancel()
	}()

	fmt.Println("Starting factory services...")

	// Start support service
	go factory.SupportService.Start(ctx)

	// Start supervisor
	go factory.Supervisor.Start(ctx)

	fmt.Println("Factory booted successfully!")
	fmt.Println("\nFactory is running. Press Ctrl+C to shutdown.")

	// Wait for context cancellation
	<-ctx.Done()

	fmt.Println("Factory shutdown complete.")
	return nil
}

func showStatus() error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	// Check if factory is initialized
	factoryDir := config.GetFactoryDir(absPath)
	if _, err := os.Stat(factoryDir); os.IsNotExist(err) {
		fmt.Println("Factory not initialized. Run 'factory init' first.")
		return nil
	}

	// Load config
	cfg, err := loadConfig(absPath)
	if err != nil {
		return err
	}

	// Initialize store
	dbPath := filepath.Join(absPath, cfg.Database.Path)
	s, err := store.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Get station status
	stations, err := s.ListStations(ctx)
	if err != nil {
		stations = []*store.Station{}
	}

	// Get operator status
	operators, err := s.ListOperators(ctx)
	if err != nil {
		operators = []*store.Operator{}
	}

	// Print status
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    FACTORY STATUS                         ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Factory: %-47s║\n", cfg.Factory.Name)
	fmt.Printf("║  Path: %-49s║\n", absPath)
	fmt.Printf("║  Max Stations: %-41d║\n", cfg.Factory.MaxStations)
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")

	// Station summary
	idle := 0
	busy := 0
	for _, s := range stations {
		if s.Status == string(station.StationIdle) {
			idle++
		} else if s.Status == string(station.StationBusy) {
			busy++
		}
	}

	fmt.Printf("║  Stations: %-6d (Idle: %-4d Busy: %-4d)           ║\n", len(stations), idle, busy)

	// Operator summary
	activeOps := 0
	stuckOps := 0
	for _, o := range operators {
		if o.Status == string(operator.OperatorWorking) {
			activeOps++
		} else if o.Status == string(operator.OperatorStuck) {
			stuckOps++
		}
	}

	fmt.Printf("║  Operators: %-5d (Active: %-4d Stuck: %-4d)        ║\n", len(operators), activeOps, stuckOps)
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")

	// List stations
	if len(stations) > 0 {
		fmt.Println("║  STATIONS                                                 ║")
		fmt.Println("╟───────────────────────────────────────────────────────────╢")
		for _, s := range stations {
			status := string(s.Status)
			if s.CurrentJob != "" {
				fmt.Printf("║    %-20s %-10s Job: %-20s║\n", s.Name, status, s.CurrentJob)
			} else {
				fmt.Printf("║    %-20s %-45s║\n", s.Name, status)
			}
		}
	}

	fmt.Println("╚═══════════════════════════════════════════════════════════╝")

	return nil
}

func shutdownFactory() error {
	if factoryInstance == nil {
		fmt.Println("Factory is not running.")
		return nil
	}

	fmt.Println("Shutting down factory...")

	// Director handles graceful shutdown
	if factoryInstance.Director != nil {
		if err := factoryInstance.Director.Stop(); err != nil {
			return fmt.Errorf("stopping director: %w", err)
		}
	}

	// Close store
	if factoryInstance.Store != nil {
		factoryInstance.Store.Close()
	}

	factoryInstance = nil
	fmt.Println("Factory shutdown complete.")
	return nil
}

func pauseFactory() error {
	if factoryInstance == nil {
		fmt.Println("Factory is not running.")
		return nil
	}

	if err := factoryInstance.Director.Pause(); err != nil {
		return err
	}

	fmt.Println("Factory paused.")
	return nil
}

func resumeFactory() error {
	if factoryInstance == nil {
		fmt.Println("Factory is not running.")
		return nil
	}

	if err := factoryInstance.Director.Resume(); err != nil {
		return err
	}

	fmt.Println("Factory resumed.")
	return nil
}

func loadConfig(absPath string) (*config.Config, error) {
	var cfg *config.Config
	var err error

	if cfgFile != "" {
		cfg, err = config.Load(cfgFile)
	} else {
		cfg, err = config.LoadFromDir(absPath)
	}

	if err != nil {
		// Check if factory.yaml doesn't exist
		configPath := filepath.Join(absPath, "factory.yaml")
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
			// Ask user if they want to generate a sample factory.yaml
			fmt.Println("factory.yaml not found.")
			fmt.Print("Would you like to generate a sample factory.yaml? [y/N]: ")

			var response string
			fmt.Scanln(&response)

			if response == "y" || response == "Y" || response == "yes" {
				// Generate sample config
				cfg = config.DefaultConfig()
				cfg.Factory.ProjectPath = absPath

				if saveErr := cfg.Save(configPath); saveErr != nil {
					return nil, fmt.Errorf("saving sample config: %w", saveErr)
				}

				fmt.Printf("Generated sample factory.yaml at: %s\n", configPath)
				fmt.Println("Please review and edit the configuration before booting the factory.")
				return cfg, nil
			}

			return nil, fmt.Errorf("factory.yaml not found. Run 'factory init' or generate a sample config")
		}
		return nil, fmt.Errorf("loading config: %w", err)
	}

	cfg.Factory.ProjectPath = absPath
	return cfg, nil
}

func initializeFactory(cfg *config.Config, absPath string) (*Factory, error) {
	// Initialize store
	dbPath := filepath.Join(absPath, cfg.Database.Path)
	s, err := store.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}

	if err := s.Migrate(); err != nil {
		s.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	// Initialize event bus
	eventBus := events.NewEventBus(100, s)

	// Initialize tmux manager
	tmuxManager, err := tmux.NewManager()
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("creating tmux manager: %w", err)
	}

	// Initialize beads client
	beadsClient, err := beads.NewClient(cfg.Beads.BinaryPath, absPath)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("creating beads client: %w", err)
	}

	// Initialize station manager
	stationManager := station.NewManager(absPath, eventBus, s, tmuxManager, cfg.Factory.MaxStations)

	// Initialize operator pool
	operatorPool := operator.NewPool(stationManager, eventBus, s, tmuxManager, beadsClient)

	// Initialize traveler manager
	travelerMgr := traveler.NewManager(beadsClient, eventBus, s)

	// Initialize planner (director will be set later)
	plannerInstance := planner.NewPlanner(eventBus, s, travelerMgr, stationManager, beadsClient, nil)

	// Initialize supervisor
	supervisorInstance := supervisor.NewSupervisor(eventBus, s, tmuxManager)

	// Initialize support service
	supportService := support.NewService(eventBus, s, tmuxManager)

	// Initialize director
	directorInstance := director.NewDirector(
		plannerInstance,
		stationManager,
		supervisorInstance,
		supportService,
		eventBus,
		s,
		tmuxManager,
		beadsClient,
	)

	// Initialize workflow engine
	workflowEngine := workflow.NewDAGEngine(eventBus)

	return &Factory{
		Config:         cfg,
		Store:          s,
		EventBus:       eventBus,
		TmuxManager:    tmuxManager,
		BeadsClient:    beadsClient,
		StationManager: stationManager,
		OperatorPool:   operatorPool,
		Planner:        plannerInstance,
		Supervisor:     supervisorInstance,
		SupportService: supportService,
		Director:       directorInstance,
		WorkflowEngine: workflowEngine,
	}, nil
}

// GetFactory returns the current factory instance
func GetFactory() *Factory {
	return factoryInstance
}

// SetFactory sets the factory instance (for testing)
func SetFactory(f *Factory) {
	factoryInstance = f
}

// getOrCreateFactory returns the existing factory instance or creates a new one
// This allows commands to work without requiring 'factory boot' to be running
func getOrCreateFactory() (*Factory, error) {
	// Return existing instance if available
	if factoryInstance != nil {
		return factoryInstance, nil
	}

	// Create a new factory instance from config
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path: %w", err)
	}

	// Load config
	cfg, err := loadConfig(absPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w (run 'factory init' first)", err)
	}

	// Initialize factory components
	factory, err := initializeFactory(cfg, absPath)
	if err != nil {
		return nil, fmt.Errorf("initializing factory: %w", err)
	}

	return factory, nil
}
