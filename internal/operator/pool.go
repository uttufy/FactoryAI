// Package operator manages AI operators that work at stations.
package operator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/tmux"
)

// OperatorStatus represents the status of an operator
type OperatorStatus string

const (
	OperatorIdle     OperatorStatus = "idle"
	OperatorWorking  OperatorStatus = "working"
	OperatorDone     OperatorStatus = "done"
	OperatorFailed   OperatorStatus = "failed"
	OperatorStuck    OperatorStatus = "stuck"
	OperatorHandoff  OperatorStatus = "handoff"
)

// Operator represents an AI agent working at a station
type Operator struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	StationID     string         `json:"station_id"`
	Status        OperatorStatus `json:"status"`
	CurrentTask   string         `json:"current_task,omitempty"`
	ClaudeSession string         `json:"claude_session,omitempty"`
	StartedAt     time.Time      `json:"started_at"`
	LastHeartbeat time.Time      `json:"last_heartbeat"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	Skills        []string       `json:"skills,omitempty"`
}

// Pool manages operators
type Pool struct {
	stationManager *station.Manager
	events         *events.EventBus
	store          *store.Store
	tmux           *tmux.Manager
	client         *beads.Client
	operators      map[string]*Operator
	mu             sync.RWMutex
}

// NewPool creates a new operator pool
func NewPool(
	stationManager *station.Manager,
	events *events.EventBus,
	store *store.Store,
	tmux *tmux.Manager,
	client *beads.Client,
) *Pool {
	return &Pool{
		stationManager: stationManager,
		events:         events,
		store:          store,
		tmux:           tmux,
		client:         client,
		operators:      make(map[string]*Operator),
	}
}

// Spawn creates a new operator at a station
func (p *Pool) Spawn(ctx context.Context, stationID string) (*Operator, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if station exists and is available
	st, err := p.stationManager.Get(ctx, stationID)
	if err != nil {
		return nil, fmt.Errorf("getting station: %w", err)
	}

	if st.Status != station.StationIdle {
		return nil, fmt.Errorf("station not idle: %s (status: %s)", stationID, st.Status)
	}

	now := time.Now()
	operatorID := uuid.New().String()

	operator := &Operator{
		ID:            operatorID,
		Name:          fmt.Sprintf("operator-%s", stationID),
		StationID:     stationID,
		Status:        OperatorIdle,
		StartedAt:     now,
		LastHeartbeat: now,
		Skills:        []string{"general"},
	}

	p.operators[operatorID] = operator

	// Update station
	_ = p.stationManager.SetOperator(ctx, stationID, operatorID)

	// Emit event
	p.events.Emit(events.EventOperatorSpawned, "operator_pool", operatorID, map[string]interface{}{
		"station_id": stationID,
		"name":       operator.Name,
	})

	return operator, nil
}

// SpawnWithTask creates an operator and assigns a task
func (p *Pool) SpawnWithTask(ctx context.Context, stationID, beadID string) (*Operator, error) {
	operator, err := p.Spawn(ctx, stationID)
	if err != nil {
		return nil, err
	}

	// Attach task to station
	travelerOpts := []beads.AttachOption{beads.WithPriority(0)}
	if err := p.client.AttachTraveler(stationID, beadID, travelerOpts...); err != nil {
		_ = p.Decommission(ctx, operator.ID)
		return nil, fmt.Errorf("attaching task: %w", err)
	}

	operator.Status = OperatorWorking
	operator.CurrentTask = beadID

	// Update station
	_ = p.stationManager.SetBusy(ctx, stationID, beadID)

	p.events.Emit(events.EventOperatorIdle, "operator_pool", operator.ID, map[string]interface{}{
		"task_id": beadID,
	})

	return operator, nil
}

// Get retrieves an operator by ID
func (p *Pool) Get(ctx context.Context, operatorID string) (*Operator, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return nil, fmt.Errorf("operator not found: %s", operatorID)
	}

	return operator, nil
}

// List returns all operators
func (p *Pool) List(ctx context.Context) []*Operator {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var operators []*Operator
	for _, operator := range p.operators {
		operators = append(operators, operator)
	}

	return operators
}

// Decommission gracefully stops an operator
func (p *Pool) Decommission(ctx context.Context, operatorID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	// Update station
	if operator.Status == OperatorWorking {
		_ = p.stationManager.SetIdle(ctx, operator.StationID)
	}

	now := time.Now()
	operator.CompletedAt = &now
	operator.Status = OperatorDone

	// Emit event
	p.events.Emit(events.EventOperatorIdle, "operator_pool", operatorID, map[string]interface{}{
		"station_id": operator.StationID,
	})

	// Remove from pool after a delay
	go func() {
		time.Sleep(5 * time.Minute)
		p.mu.Lock()
		delete(p.operators, operatorID)
		p.mu.Unlock()
	}()

	return nil
}

// Handoff gracefully restarts an operator with context transfer
func (p *Pool) Handoff(ctx context.Context, operatorID string, workOnTraveler bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	stationID := operator.StationID
	currentTask := operator.CurrentTask

	// Decommission old operator
	operator.Status = OperatorHandoff
	now := time.Now()
	operator.CompletedAt = &now

	// Spawn new operator
	newOperator := &Operator{
		ID:            uuid.New().String(),
		Name:          fmt.Sprintf("operator-%s", stationID),
		StationID:     stationID,
		Status:        OperatorIdle,
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Skills:        operator.Skills,
	}

	p.operators[newOperator.ID] = newOperator

	// Update station
	_ = p.stationManager.SetOperator(ctx, stationID, newOperator.ID)

	// Re-attach traveler if requested
	if workOnTraveler && currentTask != "" {
		travelerOpts := []beads.AttachOption{
			beads.WithPriority(0),
			beads.WithRestart(),
		}
		_ = p.client.AttachTraveler(stationID, currentTask, travelerOpts...)
		newOperator.Status = OperatorWorking
		newOperator.CurrentTask = currentTask
		_ = p.stationManager.SetBusy(ctx, stationID, currentTask)
	}

	// Emit handoff event
	p.events.Emit(events.EventOperatorHandoff, "operator_pool", newOperator.ID, map[string]interface{}{
		"from_operator_id": operatorID,
		"station_id":       stationID,
		"task_id":          currentTask,
	})

	// Remove old operator
	delete(p.operators, operatorID)

	return nil
}

// SendHeartbeat updates the operator's last heartbeat
func (p *Pool) SendHeartbeat(ctx context.Context, operatorID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	operator.LastHeartbeat = time.Now()
	return nil
}

// GetStuck returns operators that haven't sent heartbeat recently
func (p *Pool) GetStuck(ctx context.Context, timeout time.Duration) []*Operator {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var stuck []*Operator
	now := time.Now()

	for _, operator := range p.operators {
		if operator.Status == OperatorWorking {
			if now.Sub(operator.LastHeartbeat) > timeout {
				operator.Status = OperatorStuck
				stuck = append(stuck, operator)

				p.events.Emit(events.EventOperatorStuck, "operator_pool", operator.ID, map[string]interface{}{
					"station_id":      operator.StationID,
					"last_heartbeat":  operator.LastHeartbeat,
					"timeout_seconds": timeout.Seconds(),
				})
			}
		}
	}

	return stuck
}

// UpdateStatus updates the status of an operator
func (p *Pool) UpdateStatus(ctx context.Context, operatorID string, status OperatorStatus) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	operator.Status = status
	operator.LastHeartbeat = time.Now()

	return nil
}

// GetOperatorByStation retrieves the operator at a given station
func (p *Pool) GetOperatorByStation(ctx context.Context, stationID string) (*Operator, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, operator := range p.operators {
		if operator.StationID == stationID && operator.Status != OperatorDone {
			return operator, nil
		}
	}

	return nil, fmt.Errorf("no operator found at station: %s", stationID)
}

// MarkFailed marks an operator as failed
func (p *Pool) MarkFailed(ctx context.Context, operatorID, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	now := time.Now()
	operator.Status = OperatorFailed
	operator.CompletedAt = &now

	// Update station
	_ = p.stationManager.SetIdle(ctx, operator.StationID)

	p.events.Emit(events.EventStepFailed, "operator_pool", operatorID, map[string]interface{}{
		"station_id": operator.StationID,
		"reason":     reason,
	})

	return nil
}

// StartClaudeSession starts a claude session at a station
func (p *Pool) StartClaudeSession(ctx context.Context, operatorID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	operator, ok := p.operators[operatorID]
	if !ok {
		return fmt.Errorf("operator not found: %s", operatorID)
	}

	st, err := p.stationManager.Get(ctx, operator.StationID)
	if err != nil {
		return fmt.Errorf("getting station: %w", err)
	}

	// Start claude in tmux session
	cmd := fmt.Sprintf("cd %s && claude", st.WorktreePath)
	if err := p.tmux.SendKeys(st.TmuxSession, cmd); err != nil {
		return fmt.Errorf("starting claude: %w", err)
	}

	operator.ClaudeSession = fmt.Sprintf("claude-%s", operator.ID)
	operator.Status = OperatorWorking

	return nil
}

// RunCommand runs a command at the operator's station
func (p *Pool) RunCommand(ctx context.Context, operatorID, command string) error {
	operator, err := p.Get(ctx, operatorID)
	if err != nil {
		return err
	}

	st, err := p.stationManager.Get(ctx, operator.StationID)
	if err != nil {
		return fmt.Errorf("getting station: %w", err)
	}

	return p.tmux.SendKeys(st.TmuxSession, command)
}

// GetActiveCount returns the number of active operators
func (p *Pool) GetActiveCount(ctx context.Context) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, operator := range p.operators {
		if operator.Status == OperatorWorking {
			count++
		}
	}

	return count
}

// GetIdleCount returns the number of idle operators
func (p *Pool) GetIdleCount(ctx context.Context) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, operator := range p.operators {
		if operator.Status == OperatorIdle {
			count++
		}
	}

	return count
}
