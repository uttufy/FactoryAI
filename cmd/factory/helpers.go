package main

import (
	"fmt"
	"os"

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
	"github.com/uttufy/FactoryAI/internal/workcell"
	"github.com/uttufy/FactoryAI/internal/workflow"
)

// Global flags
var (
	configPath  string
	projectPath string
)

// Command flags
var (
	stationName    string
	batchName      string
	formulaPath    string
	priority       int
	message        string
	operatorID     string
	mrID           string
	reason         string
	stationID      string
	beadID         string
	sopID          string
	role           string
	maxStations    int
	fromStation    string
	toStation      string
	workOnTraveler bool
)

// Global state
var (
	directorInstance *director.Director
	storeInstance    *store.Store
	tmuxInstance     *tmux.Manager
	eventsInstance   *events.EventBus
	beadsClient      *beads.Client
	stationManager   *station.Manager
	operatorPool     *operator.Pool
	plannerInstance  *planner.Planner
	supervisorInst   *supervisor.Supervisor
	supportService   *support.Service
	batchMgr         *batch.Manager
	dagEngine        *workflow.DAGEngine
	workCellManager  *workcell.Manager
	travelerMgr      *traveler.Manager
	assemblyMgr      *assembly.Assembly
	mailSystem       *mail.Service
)

// Factory configuration
var (
	factoryDir     = ".factory"
	dbPath         = ".factory/factory.db"
	bootStatusPath = ".factory/booted"
)

// requireBoot checks if the factory has been initialized and booted
// Uses database-backed state to work across process boundaries
func requireBoot() error {
	// Initialize store if needed
	if storeInstance == nil {
		var err error
		storeInstance, err = store.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("factory not initialized. Run 'factory init' first")
		}
	}

	// Check database for factory state
	status, err := storeInstance.GetFactoryStatus()
	if err != nil {
		return fmt.Errorf("checking factory status: %w", err)
	}

	switch status.BootStatus {
	case "running":
		return nil
	case "booting":
		return fmt.Errorf("factory is booting. Wait a moment and try again")
	case "shutting_down":
		return fmt.Errorf("factory is shutting down. Wait and try again")
	case "stopped":
		// Check if there's a stale boot status file (legacy cleanup)
		if _, err := os.Stat(bootStatusPath); err == nil {
			os.Remove(bootStatusPath)
		}
		return fmt.Errorf("factory not booted. Run 'factory boot' first")
	default:
		return fmt.Errorf("unknown factory status: %s", status.BootStatus)
	}
}

// getStore returns the store instance, initializing it if necessary
func getStore() (*store.Store, error) {
	if storeInstance == nil {
		var err error
		storeInstance, err = store.NewStore(dbPath)
		if err != nil {
			return nil, fmt.Errorf("factory not initialized. Run 'factory init' first")
		}
	}
	return storeInstance, nil
}

// markFactoryBooted creates a status file to indicate boot is in progress/complete
func markFactoryBooted() error {
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(bootStatusPath, []byte("booted"), 0644)
}

// markFactoryShutdown removes the boot status file
func markFactoryShutdown() error {
	return os.Remove(bootStatusPath)
}

// printSuccess prints a success message with a checkmark
func printSuccess(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// printError prints an error message with a cross
func printError(format string, args ...interface{}) {
	fmt.Printf("✗ "+format+"\n", args...)
}

// printInfo prints an info message
func printInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ "+format+"\n", args...)
}

// consoleEventLogger implements events.EventLogger for console output
type consoleEventLogger struct{}

func (l *consoleEventLogger) LogEvent(event events.Event) error {
	fmt.Printf("[EVENT] %s: %s\n", event.Type, event.Subject)
	return nil
}

// shutdownDirector gracefully shuts down the director
func shutdownDirector() error {
	// Clean up boot status file (legacy)
	defer markFactoryShutdown()

	// Update database state
	if st, err := getStore(); err == nil {
		st.SetFactoryStopped()
	}

	if directorInstance != nil {
		return directorInstance.Stop()
	}
	return nil
}
