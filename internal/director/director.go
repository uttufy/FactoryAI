// Package director implements the Plant Director - the top-level orchestrator and single authority.
package director

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/batch"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/operator"
	"github.com/uttufy/FactoryAI/internal/planner"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/support"
	"github.com/uttufy/FactoryAI/internal/supervisor"
	"github.com/uttufy/FactoryAI/internal/tmux"
	"github.com/uttufy/FactoryAI/internal/workflow"
)

// FactoryStatus represents the current status of the factory
type FactoryStatus struct {
	Running         bool                 `json:"running"`
	Stations        []StationStatusInfo  `json:"stations"`
	ActiveJobs      int                  `json:"active_jobs"`
	PendingBatches  int                  `json:"pending_batches"`
	LastActivity    time.Time            `json:"last_activity"`
	Uptime          time.Duration        `json:"uptime"`
}

// StationStatusInfo represents status info for a station
type StationStatusInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	CurrentJob   string `json:"current_job,omitempty"`
	OperatorID   string `json:"operator_id,omitempty"`
}

// Director is the Plant Director - top-level orchestrator
type Director struct {
	planner        *planner.Planner
	stationManager *station.Manager
	supervisor     *supervisor.Supervisor
	supportService *support.Service
	events         *events.EventBus
	store          *store.Store
	tmux           *tmux.Manager
	client         *beads.Client
	startedAt      time.Time
	mu             sync.RWMutex
	running        bool
	paused         bool
}

// NewDirector creates a new Plant Director
func NewDirector(
	planner *planner.Planner,
	stationManager *station.Manager,
	supervisor *supervisor.Supervisor,
	supportService *support.Service,
	events *events.EventBus,
	store *store.Store,
	tmux *tmux.Manager,
	client *beads.Client,
) *Director {
	return &Director{
		planner:        planner,
		stationManager: stationManager,
		supervisor:     supervisor,
		supportService: supportService,
		events:         events,
		store:          store,
		tmux:           tmux,
		client:         client,
	}
}

// Start initializes and starts the factory
func (d *Director) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("factory already running")
	}

	d.startedAt = time.Now()
	d.running = true
	d.paused = false

	// Start all components
	if err := d.planner.Start(ctx); err != nil {
		return fmt.Errorf("starting planner: %w", err)
	}

	if err := d.supervisor.Start(ctx); err != nil {
		return fmt.Errorf("starting supervisor: %w", err)
	}

	if err := d.supportService.Start(ctx); err != nil {
		return fmt.Errorf("starting support service: %w", err)
	}

	// Setup signal handling
	go d.handleSignals(ctx)

	// Emit factory started event
	d.events.Emit(events.EventFactoryShutdown, "director", "factory", map[string]interface{}{
		"action": "started",
		"time":   d.startedAt.Unix(),
	})

	return nil
}

// Stop gracefully shuts down the factory
func (d *Director) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return fmt.Errorf("factory not running")
	}

	d.running = false

	// Emit shutdown event
	d.events.Emit(events.EventFactoryShutdown, "director", "factory", map[string]interface{}{
		"action": "stopped",
		"uptime": time.Since(d.startedAt).Seconds(),
	})

	return nil
}

// ReceiveTask receives a task from the user
func (d *Director) ReceiveTask(ctx context.Context, task string) (*batch.Batch, error) {
	// Create a bead for the task
	bead, err := d.client.Create(string(beads.BeadTask), task)
	if err != nil {
		return nil, fmt.Errorf("creating task bead: %w", err)
	}

	// Create a batch with this single bead
	batchMgr := batch.NewManager(d.client, d.events, d.store)
	resultBatch, err := batchMgr.Create(task, []string{bead.ID})
	if err != nil {
		return nil, fmt.Errorf("creating batch: %w", err)
	}

	// Enqueue for processing
	if err := d.planner.Enqueue(ctx, bead.ID, 0); err != nil {
		return nil, fmt.Errorf("enqueuing task: %w", err)
	}

	return resultBatch, nil
}

// GetStatus returns current factory status
func (d *Director) GetStatus(ctx context.Context) (*FactoryStatus, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stations := d.stationManager.List(ctx)
	status := &FactoryStatus{
		Running:      d.running,
		Stations:     make([]StationStatusInfo, len(stations)),
		ActiveJobs:   d.planner.GetActiveCount(ctx),
		LastActivity: time.Now(),
		Uptime:       time.Since(d.startedAt),
	}

	for i, st := range stations {
		status.Stations[i] = StationStatusInfo{
			ID:         st.ID,
			Name:       st.Name,
			Status:     string(st.Status),
			CurrentJob: st.CurrentJob,
			OperatorID: st.OperatorID,
		}
	}

	return status, nil
}

// RunBatch creates and runs a production batch
func (d *Director) RunBatch(ctx context.Context, name string, beadIDs []string) (*batch.Batch, error) {
	// Create batch
	batchMgr := batch.NewManager(d.client, d.events, d.store)
	resultBatch, err := batchMgr.Create(name, beadIDs)
	if err != nil {
		return nil, fmt.Errorf("creating batch: %w", err)
	}

	// Enqueue all beads
	for _, beadID := range beadIDs {
		if err := d.planner.Enqueue(ctx, beadID, 0); err != nil {
			return nil, fmt.Errorf("enqueuing bead %s: %w", beadID, err)
		}
	}

	return resultBatch, nil
}

// CreateBatch creates a batch without starting it
func (d *Director) CreateBatch(ctx context.Context, name string, beadIDs []string) (*batch.Batch, error) {
	batchMgr := batch.NewManager(d.client, d.events, d.store)
	return batchMgr.Create(name, beadIDs)
}

// Pause pauses the factory (only Director can do this)
func (d *Director) Pause() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return fmt.Errorf("factory not running")
	}

	d.paused = true

	d.events.Emit(events.EventHealthOK, "director", "factory", map[string]interface{}{
		"action": "paused",
	})

	return nil
}

// Resume resumes the factory
func (d *Director) Resume() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return fmt.Errorf("factory not running")
	}

	d.paused = false

	d.events.Emit(events.EventHealthOK, "director", "factory", map[string]interface{}{
		"action": "resumed",
	})

	return nil
}

// Escalate escalates an issue to human attention
func (d *Director) Escalate(issue string) error {
	return d.supervisor.Escalate(context.Background(), issue)
}

// ProvisionStation provisions a new station
func (d *Director) ProvisionStation(ctx context.Context, name string) (*station.Station, error) {
	return d.stationManager.Provision(ctx, name)
}

// DecommissionStation decommissions a station
func (d *Director) DecommissionStation(ctx context.Context, stationID string) error {
	return d.stationManager.Decommission(ctx, stationID)
}

// SpawnOperator spawns an operator at a station
func (d *Director) SpawnOperator(ctx context.Context, stationID string) (*operator.Operator, error) {
	// Need to create operator pool - for now return error
	return nil, fmt.Errorf("operator pool not yet integrated")
}

// CreateSOP creates a new SOP from steps
func (d *Director) CreateSOP(ctx context.Context, name string, steps []*workflow.Step) (*workflow.SOP, error) {
	// Create a DAG engine for SOP management
	dagEngine := workflow.NewDAGEngine(d.events)
	return dagEngine.CreateSOP(name, steps)
}

// RunFormula runs a formula template
func (d *Director) RunFormula(ctx context.Context, formulaPath string, vars map[string]string, task string) (*workflow.SOP, error) {
	// Load formula
	formula, err := workflow.LoadFormula(formulaPath)
	if err != nil {
		return nil, fmt.Errorf("loading formula: %w", err)
	}

	// Cook formula
	protomolecule, err := formula.CookWithVars(vars)
	if err != nil {
		return nil, fmt.Errorf("cooking formula: %w", err)
	}

	// Instantiate SOP
	sop, err := protomolecule.Instantiate(vars)
	if err != nil {
		return nil, fmt.Errorf("instantiating SOP: %w", err)
	}

	// Create task bead
	bead, err := d.client.Create(string(beads.BeadTask), task)
	if err != nil {
		return nil, fmt.Errorf("creating task bead: %w", err)
	}

	// Enqueue first step
	if len(sop.Steps) > 0 {
		if err := d.planner.Enqueue(ctx, bead.ID, 0); err != nil {
			return nil, fmt.Errorf("enqueuing first step: %w", err)
		}
	}

	return sop, nil
}

// handleSignals handles OS signals
func (d *Director) handleSignals(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigChan:
			_ = fmt.Sprintf("Received signal: %v", sig)
			_ = d.Stop()
		}
	}
}

// IsRunning returns whether the factory is running
func (d *Director) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// IsPaused returns whether the factory is paused
func (d *Director) IsPaused() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.paused
}

// GetUptime returns the factory uptime
func (d *Director) GetUptime() time.Duration {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return time.Since(d.startedAt)
}

// NudgeOperator nudges an operator
func (d *Director) NudgeOperator(ctx context.Context, operatorID, message string) error {
	return d.supportService.Nudge(ctx, operatorID, message)
}

// RunHealthCheck runs a health check
func (d *Director) RunHealthCheck(ctx context.Context) (*support.HealthReport, error) {
	return d.supportService.RunHealthCheck(ctx)
}

// GetSupervisorStatus returns floor status
func (d *Director) GetSupervisorStatus(ctx context.Context) (*supervisor.FloorStatus, error) {
	// Would need operator pool - return basic status for now
	status := &supervisor.FloorStatus{
		TotalStations: len(d.stationManager.List(ctx)),
		LastActivity:  time.Now(),
	}

	return status, nil
}
