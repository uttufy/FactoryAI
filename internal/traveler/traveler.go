// Package traveler manages Travelers (work orders) that move through stations.
package traveler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/store"
)

// Manager manages travelers (work orders) for stations
type Manager struct {
	client *beads.Client
	events *events.EventBus
	store  *store.Store
}

// NewManager creates a new traveler manager
func NewManager(client *beads.Client, eventBus *events.EventBus, store *store.Store) *Manager {
	return &Manager{
		client: client,
		events: eventBus,
		store:  store,
	}
}

// Attach assigns work to a station
func (t *Manager) Attach(ctx context.Context, stationID, beadID string, opts ...beads.AttachOption) error {
	now := time.Now()

	traveler := &beads.Traveler{
		ID:        uuid.New().String(),
		StationID: stationID,
		BeadID:    beadID,
		Status:    beads.TravelerPending,
		Priority:  0,
		AttachedAt: now,
	}

	// Apply options
	for _, opt := range opts {
		opt(traveler)
	}

	// Attach via beads CLI
	if err := t.client.AttachTraveler(stationID, beadID, opts...); err != nil {
		return fmt.Errorf("attaching traveler: %w", err)
	}

	// Emit event
	t.events.Emit(events.EventStepQueued, "traveler_manager", traveler.ID, map[string]interface{}{
		"station_id":    stationID,
		"bead_id":       beadID,
		"priority":      traveler.Priority,
		"deferred":      traveler.Deferred,
		"restart":       traveler.Restart,
	})

	return nil
}

// AttachSOP assigns a molecule/SOP to a station
func (t *Manager) AttachSOP(ctx context.Context, stationID, sopID string, opts ...beads.AttachOption) error {
	// For SOPs, we create a special bead type
	now := time.Now()

	traveler := &beads.Traveler{
		ID:        uuid.New().String(),
		StationID: stationID,
		BeadID:    sopID, // Using SOP ID as bead ID for now
		SOPID:     sopID,
		Status:    beads.TravelerPending,
		Priority:  0,
		AttachedAt: now,
	}

	// Apply options
	for _, opt := range opts {
		opt(traveler)
	}

	// Create a SOP bead and attach it
	bead, err := t.client.Create(string(beads.BeadSOP), fmt.Sprintf("SOP: %s", sopID))
	if err != nil {
		return fmt.Errorf("creating SOP bead: %w", err)
	}

	traveler.BeadID = bead.ID

	// Attach via beads CLI
	if err := t.client.AttachTraveler(stationID, bead.ID, opts...); err != nil {
		return fmt.Errorf("attaching SOP traveler: %w", err)
	}

	// Emit event
	t.events.Emit(events.EventStepQueued, "traveler_manager", traveler.ID, map[string]interface{}{
		"station_id": stationID,
		"sop_id":     sopID,
		"bead_id":    bead.ID,
		"priority":   traveler.Priority,
	})

	return nil
}

// Detach removes work from a station
func (t *Manager) Detach(ctx context.Context, stationID string) error {
	if err := t.client.ClearTraveler(stationID); err != nil {
		return fmt.Errorf("clearing traveler: %w", err)
	}

	t.events.Emit(events.EventStepCompleted, "traveler_manager", stationID, map[string]interface{}{
		"station_id": stationID,
	})

	return nil
}

// GetTraveler retrieves the current traveler for a station
func (t *Manager) GetTraveler(ctx context.Context, stationID string) (*beads.Traveler, error) {
	traveler, err := t.client.GetTraveler(stationID)
	if err != nil {
		return nil, fmt.Errorf("getting traveler: %w", err)
	}

	return traveler, nil
}

// Complete marks the traveler as complete
func (t *Manager) Complete(ctx context.Context, stationID, result string) error {
	traveler, err := t.client.GetTraveler(stationID)
	if err != nil {
		return fmt.Errorf("getting traveler: %w", err)
	}

	now := time.Now()
	traveler.Status = beads.TravelerComplete
	traveler.Result = result
	traveler.CompletedAt = &now

	// Mark the bead as done
	if err := t.client.Close(traveler.BeadID); err != nil {
		return fmt.Errorf("closing bead: %w", err)
	}

	// Clear traveler from station
	if err := t.client.ClearTraveler(stationID); err != nil {
		return fmt.Errorf("clearing traveler: %w", err)
	}

	t.events.Emit(events.EventStepCompleted, "traveler_manager", traveler.ID, map[string]interface{}{
		"station_id": stationID,
		"bead_id":    traveler.BeadID,
		"result":     result,
	})

	return nil
}

// Fail marks the traveler as failed
func (t *Manager) Fail(ctx context.Context, stationID, reason string) error {
	traveler, err := t.client.GetTraveler(stationID)
	if err != nil {
		return fmt.Errorf("getting traveler: %w", err)
	}

	now := time.Now()
	traveler.Status = beads.TravelerFailed
	traveler.Error = reason
	traveler.CompletedAt = &now

	// Mark the bead as failed
	bead, err := t.client.Get(traveler.BeadID)
	if err == nil {
		bead.MarkFailed(reason)
	}

	t.events.Emit(events.EventStepFailed, "traveler_manager", traveler.ID, map[string]interface{}{
		"station_id": stationID,
		"bead_id":    traveler.BeadID,
		"reason":     reason,
	})

	return nil
}

// Rework sends the traveler back to a station with feedback
func (t *Manager) Rework(ctx context.Context, stationID, reason string) error {
	traveler, err := t.client.GetTraveler(stationID)
	if err != nil {
		return fmt.Errorf("getting traveler: %w", err)
	}

	traveler.Status = beads.TravelerRework
	traveler.ReworkCount++
	traveler.ReworkReason = reason

	t.events.Emit(events.EventReworkNeeded, "traveler_manager", traveler.ID, map[string]interface{}{
		"station_id":    stationID,
		"bead_id":       traveler.BeadID,
		"rework_count":  traveler.ReworkCount,
		"rework_reason": reason,
	})

	return nil
}

// UpdateStatus updates the status of a traveler
func (t *Manager) UpdateStatus(ctx context.Context, stationID string, status beads.TravelerStatus) error {
	traveler, err := t.client.GetTraveler(stationID)
	if err != nil {
		return fmt.Errorf("getting traveler: %w", err)
	}

	traveler.Status = status

	if status == beads.TravelerActive && traveler.StartedAt == nil {
		now := time.Now()
		traveler.StartedAt = &now
	}

	return nil
}

// ListByStatus lists all travelers with a given status
func (t *Manager) ListByStatus(ctx context.Context, status beads.TravelerStatus) ([]*beads.Traveler, error) {
	// This would require beads CLI to support filtering by traveler status
	// For now, return empty list
	return []*beads.Traveler{}, nil
}

// GetActiveTravelers returns all active travelers
func (t *Manager) GetActiveTravelers(ctx context.Context) ([]*beads.Traveler, error) {
	return t.ListByStatus(ctx, beads.TravelerActive)
}

// GetStuckTravelers returns travelers that have been active too long
func (t *Manager) GetStuckTravelers(ctx context.Context, timeout time.Duration) ([]*beads.Traveler, error) {
	active, err := t.GetActiveTravelers(ctx)
	if err != nil {
		return nil, err
	}

	var stuck []*beads.Traveler
	now := time.Now()

	for _, traveler := range active {
		if traveler.StartedAt != nil {
			if now.Sub(*traveler.StartedAt) > timeout {
				stuck = append(stuck, traveler)
			}
		}
	}

	return stuck, nil
}
