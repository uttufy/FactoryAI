// Package workflow implements the DAG workflow engine for FactoryAI.
// This includes SOPs, Steps, and dependency evaluation.
package workflow

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/events"
)

// StepStatus tracks step execution state
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepQueued     StepStatus = "queued"      // Ready to run (deps met)
	StepRunning    StepStatus = "running"
	StepWaiting    StepStatus = "waiting"     // Waiting for deps
	StepDone       StepStatus = "done"
	StepFailed     StepStatus = "failed"
	StepSkipped    StepStatus = "skipped"
)

// Step represents a single operation in a workflow
type Step struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Assignee     string       `json:"assignee,omitempty"`     // Preferred station type
	Dependencies []string     `json:"dependencies,omitempty"` // Step IDs this depends on
	Status       StepStatus   `json:"status"`
	Acceptance   string       `json:"acceptance,omitempty"`   // Acceptance criteria
	Gate         string       `json:"gate,omitempty"`         // Must pass before proceeding
	Timeout      int          `json:"timeout,omitempty"`      // Seconds
	MaxRetries   int          `json:"max_retries,omitempty"`
	Retries      int          `json:"retries"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	Result       string       `json:"result,omitempty"`
	Error        string       `json:"error,omitempty"`
}

// SOPStatus represents the status of a Standard Operating Procedure
type SOPStatus string

const (
	SOPPending  SOPStatus = "pending"
	SOPRunning  SOPStatus = "running"
	SOPComplete SOPStatus = "complete"
	SOPFailed   SOPStatus = "failed"
	SOPPaused   SOPStatus = "paused"
)

// SOP (Standard Operating Procedure) represents a workflow
type SOP struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Steps       []*Step     `json:"steps"`
	Status      SOPStatus   `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	IsWisp      bool        `json:"is_wisp,omitempty"`
}

// DAGEngine evaluates dependencies and determines what can run
type DAGEngine struct {
	events *events.EventBus
	mu     sync.RWMutex
	sops   map[string]*SOP
}

// NewDAGEngine creates a new DAG workflow engine
func NewDAGEngine(eventBus *events.EventBus) *DAGEngine {
	return &DAGEngine{
		events: eventBus,
		sops:   make(map[string]*SOP),
	}
}

// CreateSOP creates a new SOP from steps
func (e *DAGEngine) CreateSOP(name string, steps []*Step) (*SOP, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Generate UUID for SOP
	id := uuid.New().String()

	// Ensure all steps have IDs
	for i, step := range steps {
		if step.ID == "" {
			step.ID = fmt.Sprintf("%s-step-%d", id, i)
		}
		if step.Status == "" {
			step.Status = StepPending
		}
	}

	sop := &SOP{
		ID:        id,
		Name:      name,
		Steps:     steps,
		Status:    SOPPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	e.sops[id] = sop

	// Emit SOP created event
	e.events.Emit(events.EventJobCreated, "dag_engine", id, map[string]interface{}{
		"name":  name,
		"steps": len(steps),
	})

	return sop, nil
}

// GetSOP retrieves an SOP by ID
func (e *DAGEngine) GetSOP(id string) (*SOP, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sop, ok := e.sops[id]
	if !ok {
		return nil, fmt.Errorf("SOP not found: %s", id)
	}

	return sop, nil
}

// Evaluate returns all steps that are ready to run (dependencies met)
func (e *DAGEngine) Evaluate(sopID string) ([]*Step, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return nil, fmt.Errorf("SOP not found: %s", sopID)
	}

	var readySteps []*Step

	// Build a map of step statuses for quick lookup
	stepStatuses := make(map[string]StepStatus)
	for _, step := range sop.Steps {
		stepStatuses[step.ID] = step.Status
	}

	// Check each step
	for _, step := range sop.Steps {
		// Skip steps that are already done, running, or queued
		if step.Status == StepDone || step.Status == StepRunning || step.Status == StepQueued {
			continue
		}

		// Check if all dependencies are met
		depsMet := true
		for _, depID := range step.Dependencies {
			depStatus, ok := stepStatuses[depID]
			if !ok || depStatus != StepDone {
				depsMet = false
				step.Status = StepWaiting
				break
			}
		}

		if depsMet && step.Status == StepPending {
			readySteps = append(readySteps, step)
		}
	}

	return readySteps, nil
}

// QueueReady finds all ready steps and emits StepQueued events
func (e *DAGEngine) QueueReady(sopID string) error {
	readySteps, err := e.Evaluate(sopID)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, step := range readySteps {
		step.Status = StepQueued
		e.events.Emit(events.EventStepQueued, "dag_engine", step.ID, map[string]interface{}{
			"sop_id":   sopID,
			"step_id":  step.ID,
			"step_name": step.Name,
		})
	}

	// Update SOP status if we have queued steps
	sop := e.sops[sopID]
	if len(readySteps) > 0 && sop.Status == SOPPending {
		sop.Status = SOPRunning
		sop.UpdatedAt = time.Now()
		e.events.Emit(events.EventJobStarted, "dag_engine", sopID, map[string]interface{}{
			"name": sop.Name,
		})
	}

	return nil
}

// StartStep marks a step as running
func (e *DAGEngine) StartStep(sopID, stepID, stationID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	var step *Step
	for _, s := range sop.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepID)
	}

	if step.Status != StepQueued {
		return fmt.Errorf("step not queued: %s (status: %s)", stepID, step.Status)
	}

	now := time.Now()
	step.Status = StepRunning
	step.StartedAt = &now
	sop.UpdatedAt = now

	e.events.Emit(events.EventStepStarted, "dag_engine", stepID, map[string]interface{}{
		"sop_id":     sopID,
		"step_id":    stepID,
		"station_id": stationID,
	})

	return nil
}

// CompleteStep marks a step done and triggers dependency evaluation
func (e *DAGEngine) CompleteStep(sopID, stepID, result string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	var step *Step
	for _, s := range sop.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepID)
	}

	if step.Status != StepRunning {
		return fmt.Errorf("step not running: %s (status: %s)", stepID, step.Status)
	}

	now := time.Now()
	step.Status = StepDone
	step.Result = result
	step.CompletedAt = &now
	sop.UpdatedAt = now

	e.events.Emit(events.EventStepCompleted, "dag_engine", stepID, map[string]interface{}{
		"sop_id":  sopID,
		"step_id": stepID,
		"result":  result,
	})

	// Check if all steps are done
	allDone := true
	for _, s := range sop.Steps {
		if s.Status != StepDone {
			allDone = false
			break
		}
	}

	if allDone {
		sop.Status = SOPComplete
		sop.CompletedAt = &now
		e.events.Emit(events.EventJobCompleted, "dag_engine", sopID, map[string]interface{}{
			"name": sop.Name,
		})
	}

	// Queue any newly ready steps
	go e.QueueReady(sopID)

	return nil
}

// FailStep marks a step failed
func (e *DAGEngine) FailStep(sopID, stepID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	var step *Step
	for _, s := range sop.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepID)
	}

	now := time.Now()
	step.Status = StepFailed
	step.Error = reason
	step.CompletedAt = &now
	sop.UpdatedAt = now

	// Check if should retry
	if step.Retries < step.MaxRetries {
		step.Retries++
		step.Status = StepPending
		e.events.Emit(events.EventStepFailed, "dag_engine", stepID, map[string]interface{}{
			"sop_id":  sopID,
			"step_id": stepID,
			"reason":  reason,
			"retry":   step.Retries,
		})
		return nil
	}

	// Max retries exceeded
	sop.Status = SOPFailed
	e.events.Emit(events.EventJobFailed, "dag_engine", sopID, map[string]interface{}{
		"name":   sop.Name,
		"reason": reason,
	})

	return nil
}

// RetryStep retries a failed step
func (e *DAGEngine) RetryStep(sopID, stepID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	var step *Step
	for _, s := range sop.Steps {
		if s.ID == stepID {
			step = s
			break
		}
	}

	if step == nil {
		return fmt.Errorf("step not found: %s", stepID)
	}

	if step.Status != StepFailed {
		return fmt.Errorf("step not failed: %s (status: %s)", stepID, step.Status)
	}

	step.Status = StepPending
	step.Error = ""
	sop.UpdatedAt = time.Now()

	return nil
}

// GetRunningSteps returns all currently running steps
func (e *DAGEngine) GetRunningSteps(sopID string) ([]*Step, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return nil, fmt.Errorf("SOP not found: %s", sopID)
	}

	var running []*Step
	for _, step := range sop.Steps {
		if step.Status == StepRunning {
			running = append(running, step)
		}
	}

	return running, nil
}

// GetPendingSteps returns all pending steps (deps not met)
func (e *DAGEngine) GetPendingSteps(sopID string) ([]*Step, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return nil, fmt.Errorf("SOP not found: %s", sopID)
	}

	var pending []*Step
	for _, step := range sop.Steps {
		if step.Status == StepPending || step.Status == StepWaiting {
			pending = append(pending, step)
		}
	}

	return pending, nil
}

// IsComplete checks if all steps are done
func (e *DAGEngine) IsComplete(sopID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return false, fmt.Errorf("SOP not found: %s", sopID)
	}

	for _, step := range sop.Steps {
		if step.Status != StepDone {
			return false, nil
		}
	}

	return true, nil
}

// Pause pauses an SOP
func (e *DAGEngine) Pause(sopID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	if sop.Status != SOPRunning {
		return fmt.Errorf("cannot pause SOP in status: %s", sop.Status)
	}

	sop.Status = SOPPaused
	sop.UpdatedAt = time.Now()

	return nil
}

// Resume resumes a paused SOP
func (e *DAGEngine) Resume(sopID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	sop, ok := e.sops[sopID]
	if !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	if sop.Status != SOPPaused {
		return fmt.Errorf("cannot resume SOP in status: %s", sop.Status)
	}

	sop.Status = SOPRunning
	sop.UpdatedAt = time.Now()

	// Queue ready steps
	go e.QueueReady(sopID)

	return nil
}

// DeleteSOP deletes an SOP from the engine
func (e *DAGEngine) DeleteSOP(sopID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.sops[sopID]; !ok {
		return fmt.Errorf("SOP not found: %s", sopID)
	}

	delete(e.sops, sopID)
	return nil
}

// ListSOPs returns all SOPs
func (e *DAGEngine) ListSOPs() []*SOP {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var sops []*SOP
	for _, sop := range e.sops {
		sops = append(sops, sop)
	}

	return sops
}
