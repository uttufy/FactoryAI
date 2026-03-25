// Package workcell manages Work Cells - parallel execution groups.
package workcell

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/planner"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
)

// WorkCellStatus represents the status of a work cell
type WorkCellStatus string

const (
	WorkCellStaging  WorkCellStatus = "staging"
	WorkCellActive   WorkCellStatus = "active"
	WorkCellComplete WorkCellStatus = "complete"
	WorkCellFailed   WorkCellStatus = "failed"
)

// WorkCell represents a group of stations working in parallel
type WorkCell struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Status       WorkCellStatus `json:"status"`
	Stations     []string       `json:"stations"`
	TargetBeads  []string       `json:"target_beads"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	mu           sync.RWMutex
}

// Manager manages work cells
type Manager struct {
	stationManager *station.Manager
	planner        *planner.Planner
	events         *events.EventBus
	store          *store.Store
	cells          map[string]*WorkCell
	mu             sync.RWMutex
}

// NewManager creates a new work cell manager
func NewManager(
	stationManager *station.Manager,
	planner *planner.Planner,
	events *events.EventBus,
	store *store.Store,
) *Manager {
	return &Manager{
		stationManager: stationManager,
		planner:        planner,
		events:         events,
		store:          store,
		cells:          make(map[string]*WorkCell),
	}
}

// Create creates a new work cell with specified stations
func (m *Manager) Create(ctx context.Context, name string, stationIDs []string, beadIDs []string) (*WorkCell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cellID := uuid.New().String()

	// Verify all stations exist
	for _, stationID := range stationIDs {
		if _, err := m.stationManager.Get(ctx, stationID); err != nil {
			return nil, fmt.Errorf("station not found: %s", stationID)
		}
	}

	cell := &WorkCell{
		ID:          cellID,
		Name:        name,
		Status:      WorkCellStaging,
		Stations:    stationIDs,
		TargetBeads: beadIDs,
		CreatedAt:   time.Now(),
	}

	m.cells[cellID] = cell

	m.events.Emit(events.EventJobCreated, "workcell_manager", cellID, map[string]interface{}{
		"name":        name,
		"stations":    len(stationIDs),
		"target_beads": len(beadIDs),
	})

	return cell, nil
}

// Activate starts parallel execution on all stations
func (m *Manager) Activate(ctx context.Context, cellID string) error {
	m.mu.Lock()
	cell, ok := m.cells[cellID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("work cell not found: %s", cellID)
	}

	cell.mu.Lock()
	defer cell.mu.Unlock()

	if cell.Status != WorkCellStaging {
		return fmt.Errorf("work cell not in staging status: %s", cell.Status)
	}

	// Check if we have enough stations for the beads
	if len(cell.Stations) < len(cell.TargetBeads) {
		return fmt.Errorf("not enough stations: have %d, need %d", len(cell.Stations), len(cell.TargetBeads))
	}

	// Assign beads to stations and dispatch
	dispatchMap := make(map[string]string)
	for i, beadID := range cell.TargetBeads {
		if i < len(cell.Stations) {
			stationID := cell.Stations[i]
			dispatchMap[beadID] = stationID
		}
	}

	// Dispatch all beads
	if _, err := m.planner.DispatchBatch(ctx, cell.TargetBeads); err != nil {
		return fmt.Errorf("dispatching batch: %w", err)
	}

	now := time.Now()
	cell.Status = WorkCellActive
	cell.StartedAt = &now

	m.events.Emit(events.EventJobStarted, "workcell_manager", cellID, map[string]interface{}{
		"stations":    cell.Stations,
		"target_beads": cell.TargetBeads,
	})

	return nil
}

// Status returns current status of a work cell
func (m *Manager) Status(ctx context.Context, cellID string) (*WorkCell, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cell, ok := m.cells[cellID]
	if !ok {
		return nil, fmt.Errorf("work cell not found: %s", cellID)
	}

	return cell, nil
}

// Disperse stops all stations in the cell
func (m *Manager) Disperse(ctx context.Context, cellID string) error {
	m.mu.Lock()
	cell, ok := m.cells[cellID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("work cell not found: %s", cellID)
	}
	m.mu.Unlock()

	cell.mu.Lock()
	defer cell.mu.Unlock()

	if cell.Status != WorkCellActive {
		return fmt.Errorf("work cell not active: %s", cell.Status)
	}

	// Mark all stations as idle
	for _, stationID := range cell.Stations {
		_ = m.stationManager.SetIdle(ctx, stationID)
	}

	now := time.Now()
	cell.Status = WorkCellComplete
	cell.CompletedAt = &now

	m.events.Emit(events.EventJobCompleted, "workcell_manager", cellID, map[string]interface{}{
		"stations": cell.Stations,
	})

	return nil
}

// WaitForComplete blocks until all work is done or context cancelled
func (m *Manager) WaitForComplete(ctx context.Context, cellID string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cell, err := m.Status(ctx, cellID)
			if err != nil {
				return err
			}

			if cell.Status == WorkCellComplete {
				return nil
			}

			if cell.Status == WorkCellFailed {
				return fmt.Errorf("work cell failed: %s", cellID)
			}
		}
	}
}

// List returns all work cells
func (m *Manager) List(ctx context.Context) []*WorkCell {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var cells []*WorkCell
	for _, cell := range m.cells {
		cells = append(cells, cell)
	}

	return cells
}

// GetActiveCells returns all active work cells
func (m *Manager) GetActiveCells(ctx context.Context) []*WorkCell {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*WorkCell
	for _, cell := range m.cells {
		if cell.Status == WorkCellActive {
			active = append(active, cell)
		}
	}

	return active
}

// MarkFailed marks a work cell as failed
func (m *Manager) MarkFailed(ctx context.Context, cellID, reason string) error {
	m.mu.Lock()
	cell, ok := m.cells[cellID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("work cell not found: %s", cellID)
	}
	m.mu.Unlock()

	cell.mu.Lock()
	defer cell.mu.Unlock()

	now := time.Now()
	cell.Status = WorkCellFailed
	cell.CompletedAt = &now

	m.events.Emit(events.EventJobFailed, "workcell_manager", cellID, map[string]interface{}{
		"reason": reason,
	})

	return nil
}

// GetProgress returns the progress of a work cell
func (m *Manager) GetProgress(ctx context.Context, cellID string) (int, int, error) {
	cell, err := m.Status(ctx, cellID)
	if err != nil {
		return 0, 0, err
	}

	cell.mu.RLock()
	defer cell.mu.RUnlock()

	if cell.Status == WorkCellStaging {
		return 0, len(cell.TargetBeads), nil
	}

	// Count completed beads
	// This would need to check bead status via beads client
	completed := 0
	total := len(cell.TargetBeads)

	return completed, total, nil
}

// AddStations adds stations to an existing work cell
func (m *Manager) AddStations(ctx context.Context, cellID string, stationIDs []string) error {
	m.mu.Lock()
	cell, ok := m.cells[cellID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("work cell not found: %s", cellID)
	}

	if cell.Status != WorkCellStaging {
		m.mu.Unlock()
		return fmt.Errorf("cannot add stations to active cell: %s", cellID)
	}
	m.mu.Unlock()

	cell.mu.Lock()
	defer cell.mu.Unlock()

	// Verify all stations exist
	for _, stationID := range stationIDs {
		if _, err := m.stationManager.Get(ctx, stationID); err != nil {
			return fmt.Errorf("station not found: %s", stationID)
		}
	}

	cell.Stations = append(cell.Stations, stationIDs...)

	return nil
}

// RemoveCell removes a work cell
func (m *Manager) RemoveCell(ctx context.Context, cellID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.cells[cellID]; !ok {
		return fmt.Errorf("work cell not found: %s", cellID)
	}

	delete(m.cells, cellID)

	return nil
}

// GetCellByName finds a work cell by name
func (m *Manager) GetCellByName(ctx context.Context, name string) (*WorkCell, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cell := range m.cells {
		if cell.Name == name {
			return cell, nil
		}
	}

	return nil, fmt.Errorf("work cell not found: %s", name)
}
