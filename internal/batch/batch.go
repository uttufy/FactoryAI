// Package batch manages production batches.
package batch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/store"
)

// BatchStatus represents the status of a batch
type BatchStatus string

const (
	BatchStaging  BatchStatus = "staging"
	BatchRunning  BatchStatus = "running"
	BatchComplete BatchStatus = "complete"
	BatchFailed   BatchStatus = "failed"
	BatchPartial  BatchStatus = "partial" // Some completed, some failed
)

// Batch represents a production batch
type Batch struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Status      BatchStatus  `json:"status"`
	TrackedIDs  []string     `json:"tracked_ids"`
	WorkCells   []string     `json:"work_cells,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Result      string       `json:"result,omitempty"`
	mu          sync.RWMutex
}

// BatchSummary represents a batch summary for dashboard
type BatchSummary struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	TotalJobs     int     `json:"total_jobs"`
	CompletedJobs int     `json:"completed_jobs"`
	FailedJobs    int     `json:"failed_jobs"`
	Progress      float64 `json:"progress"`
}

// Manager manages batches
type Manager struct {
	client  *beads.Client
	events  *events.EventBus
	store   *store.Store
	batches map[string]*Batch
	mu      sync.RWMutex
}

// NewManager creates a new batch manager
func NewManager(client *beads.Client, events *events.EventBus, store *store.Store) *Manager {
	return &Manager{
		client:  client,
		events:  events,
		store:   store,
		batches: make(map[string]*Batch),
	}
}

// Create creates a new batch
func (m *Manager) Create(name string, trackedIDs []string) (*Batch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	batchID := uuid.New().String()
	now := time.Now()

	batch := &Batch{
		ID:         batchID,
		Name:       name,
		Status:     BatchStaging,
		TrackedIDs: trackedIDs,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	m.batches[batchID] = batch

	m.events.Emit(events.EventJobCreated, "batch_manager", batchID, map[string]interface{}{
		"name":        name,
		"tracked_ids": trackedIDs,
	})

	return batch, nil
}

// Track returns the current state of a batch
func (m *Manager) Track(ctx context.Context, batchID string) (*Batch, error) {
	m.mu.RLock()
	batch, ok := m.batches[batchID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}

	// Update progress by checking bead statuses
	m.updateBatchProgress(ctx, batch)

	return batch, nil
}

// Complete marks a batch as complete
func (m *Manager) Complete(ctx context.Context, batchID, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	batch, ok := m.batches[batchID]
	if !ok {
		return fmt.Errorf("batch not found: %s", batchID)
	}

	batch.mu.Lock()
	defer batch.mu.Unlock()

	now := time.Now()
	batch.Status = BatchComplete
	batch.Result = result
	batch.UpdatedAt = now
	batch.CompletedAt = &now

	m.events.Emit(events.EventJobCompleted, "batch_manager", batchID, map[string]interface{}{
		"result": result,
	})

	return nil
}

// Fail marks a batch as failed
func (m *Manager) Fail(ctx context.Context, batchID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	batch, ok := m.batches[batchID]
	if !ok {
		return fmt.Errorf("batch not found: %s", batchID)
	}

	batch.mu.Lock()
	defer batch.mu.Unlock()

	now := time.Now()
	batch.Status = BatchFailed
	batch.Result = reason
	batch.UpdatedAt = now
	batch.CompletedAt = &now

	m.events.Emit(events.EventJobFailed, "batch_manager", batchID, map[string]interface{}{
		"reason": reason,
	})

	return nil
}

// List returns all batches
func (m *Manager) List(ctx context.Context, filter BatchStatus) ([]*Batch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var batches []*Batch
	for _, batch := range m.batches {
		if filter == "" || batch.Status == filter {
			batches = append(batches, batch)
		}
	}

	return batches, nil
}

// Dashboard returns batch summary for TUI
func (m *Manager) Dashboard(ctx context.Context) ([]*BatchSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []*BatchSummary

	for _, batch := range m.batches {
		batch.mu.RLock()
		totalJobs := len(batch.TrackedIDs)

		// Count completed jobs
		completedJobs := 0
		failedJobs := 0

		for _, beadID := range batch.TrackedIDs {
			bead, err := m.client.Get(beadID)
			if err == nil {
				if bead.Status == beads.StatusDone {
					completedJobs++
				} else if bead.Status == beads.StatusFailed {
					failedJobs++
				}
			}
		}

		progress := 0.0
		if totalJobs > 0 {
			progress = float64(completedJobs+failedJobs) / float64(totalJobs) * 100
		}

		summary := &BatchSummary{
			ID:            batch.ID,
			Name:          batch.Name,
			Status:        string(batch.Status),
			TotalJobs:     totalJobs,
			CompletedJobs: completedJobs,
			FailedJobs:    failedJobs,
			Progress:      progress,
		}
		batch.mu.RUnlock()

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// updateBatchProgress updates the progress of a batch based on bead statuses
func (m *Manager) updateBatchProgress(ctx context.Context, batch *Batch) {
	batch.mu.Lock()
	defer batch.mu.Unlock()

	completedCount := 0
	failedCount := 0
	pendingCount := 0

	for _, beadID := range batch.TrackedIDs {
		bead, err := m.client.Get(beadID)
		if err != nil {
			continue
		}

		switch bead.Status {
		case beads.StatusDone:
			completedCount++
		case beads.StatusFailed:
			failedCount++
		default:
			pendingCount++
		}
	}

	// Update batch status based on bead statuses
	if pendingCount == 0 {
		if failedCount == 0 {
			batch.Status = BatchComplete
		} else if completedCount == 0 {
			batch.Status = BatchFailed
		} else {
			batch.Status = BatchPartial
		}
	} else if batch.Status == BatchStaging {
		batch.Status = BatchRunning
	}

	batch.UpdatedAt = time.Now()
}

// AddWorkCell adds a work cell to a batch
func (m *Manager) AddWorkCell(ctx context.Context, batchID, workCellID string) error {
	m.mu.Lock()
	batch, ok := m.batches[batchID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("batch not found: %s", batchID)
	}
	m.mu.Unlock()

	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.Status != BatchStaging {
		return fmt.Errorf("cannot add work cell to batch in status: %s", batch.Status)
	}

	batch.WorkCells = append(batch.WorkCells, workCellID)
	batch.UpdatedAt = time.Now()

	return nil
}

// GetBatch returns a batch by ID
func (m *Manager) GetBatch(batchID string) (*Batch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	batch, ok := m.batches[batchID]
	if !ok {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}

	return batch, nil
}

// DeleteBatch deletes a batch
func (m *Manager) DeleteBatch(ctx context.Context, batchID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.batches[batchID]; !ok {
		return fmt.Errorf("batch not found: %s", batchID)
	}

	delete(m.batches, batchID)

	return nil
}

// Start starts a batch (transitions from staging to running)
func (m *Manager) Start(ctx context.Context, batchID string) error {
	m.mu.Lock()
	batch, ok := m.batches[batchID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("batch not found: %s", batchID)
	}
	m.mu.Unlock()

	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.Status != BatchStaging {
		return fmt.Errorf("batch not in staging status: %s", batch.Status)
	}

	batch.Status = BatchRunning
	batch.UpdatedAt = time.Now()

	m.events.Emit(events.EventJobStarted, "batch_manager", batchID, map[string]interface{}{
		"name": batch.Name,
	})

	return nil
}

// AddBead adds a bead to an existing batch
func (m *Manager) AddBead(ctx context.Context, batchID, beadID string) error {
	m.mu.Lock()
	batch, ok := m.batches[batchID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("batch not found: %s", batchID)
	}
	m.mu.Unlock()

	batch.mu.Lock()
	defer batch.mu.Unlock()

	if batch.Status != BatchStaging {
		return fmt.Errorf("cannot add bead to batch in status: %s", batch.Status)
	}

	batch.TrackedIDs = append(batch.TrackedIDs, beadID)
	batch.UpdatedAt = time.Now()

	return nil
}

// GetActiveBatches returns all active (running) batches
func (m *Manager) GetActiveBatches(ctx context.Context) []*Batch {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*Batch
	for _, batch := range m.batches {
		if batch.Status == BatchRunning {
			active = append(active, batch)
		}
	}

	return active
}
