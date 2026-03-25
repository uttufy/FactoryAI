package main

import (
	"fmt"

	"github.com/uttufy/FactoryAI/internal/batch"
	"github.com/uttufy/FactoryAI/internal/beads"
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
)

// Factory configuration
var (
	factoryDir = ".factory"
	dbPath     = ".factory/factory.db"
)

// requireBoot checks if the factory has been initialized and booted
func requireBoot() error {
	if storeInstance == nil {
		return fmt.Errorf("factory not initialized. Run 'factory init' first")
	}
	if directorInstance == nil {
		return fmt.Errorf("factory not booted. Run 'factory boot' first")
	}
	return nil
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
	if directorInstance != nil {
		return directorInstance.Stop()
	}
	return nil
}
