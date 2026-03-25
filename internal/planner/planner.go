// Package planner implements the Production Planner - the single dispatcher authority.
package planner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/traveler"
)

// QueueItem represents a queued work item
type QueueItem struct {
	BeadID    string    `json:"bead_id"`
	Priority  int       `json:"priority"`
	QueuedAt  time.Time `json:"queued_at"`
	Status    string    `json:"status"`
	StationID string    `json:"station_id,omitempty"`
}

// Planner is the single dispatcher for the factory
type Planner struct {
	events         *events.EventBus
	store          *store.Store
	travelerMgr    *traveler.Manager
	stationManager *station.Manager
	mu             sync.RWMutex
	queue          []*QueueItem
	director       interface{} // Will be *director.Director
	client         *beads.Client
}

// NewPlanner creates a new production planner
func NewPlanner(
	events *events.EventBus,
	store *store.Store,
	travelerMgr *traveler.Manager,
	stationManager *station.Manager,
	client *beads.Client,
	director interface{},
) *Planner {
	p := &Planner{
		events:         events,
		store:          store,
		travelerMgr:    travelerMgr,
		stationManager: stationManager,
		queue:          make([]*QueueItem, 0),
		client:         client,
		director:       director,
	}

	// Subscribe to events - commented out for now
	// events.Subscribe(events.EventJobCreated, p.handleJobQueued)
	// events.Subscribe(events.EventStepCompleted, p.handleStepCompleted)
	// events.Subscribe(events.EventStationReady, p.handleStationReady)

	return p
}

// Start begins listening for events and dispatching work
func (p *Planner) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Process any existing queued items
	go p.processQueue(ctx)

	return nil
}

// Dispatch assigns a bead to a specific station
func (p *Planner) Dispatch(ctx context.Context, beadID, stationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if bead exists
	bead, err := p.client.Get(beadID)
	if err != nil {
		return fmt.Errorf("getting bead: %w", err)
	}

	// Attach traveler to station
	if err := p.travelerMgr.Attach(ctx, stationID, beadID); err != nil {
		return fmt.Errorf("attaching traveler: %w", err)
	}

	_ = bead // Use the bead variable

	// Update queue item status
	for _, item := range p.queue {
		if item.BeadID == beadID {
			item.Status = "dispatched"
			item.StationID = stationID
			break
		}
	}

	// Emit event
	p.events.Emit(events.EventStepStarted, "planner", beadID, map[string]interface{}{
		"station_id": stationID,
		"bead_id":    beadID,
	})

	return nil
}

// AutoDispatch finds an available station for the bead
func (p *Planner) AutoDispatch(ctx context.Context, beadID string) (string, error) {
	available := p.stationManager.GetAvailable(ctx)
	if len(available) == 0 {
		return "", fmt.Errorf("no available stations")
	}

	// For now, pick the first available station
	// In future, could implement smarter scheduling based on:
	// - Station capabilities/skills
	// - Bead type/priority
	// - Load balancing
	station := available[0]

	return station.ID, p.Dispatch(ctx, beadID, station.ID)
}

// DispatchBatch dispatches multiple beads to available stations
func (p *Planner) DispatchBatch(ctx context.Context, beadIDs []string) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]string)
	available := p.stationManager.GetAvailable(ctx)

	if len(available) < len(beadIDs) {
		return result, fmt.Errorf("not enough available stations: have %d, need %d", len(available), len(beadIDs))
	}

	for i, beadID := range beadIDs {
		stationID := available[i].ID
		if err := p.Dispatch(ctx, beadID, stationID); err != nil {
			return result, fmt.Errorf("dispatching bead %s: %w", beadID, err)
		}
		result[beadID] = stationID
	}

	return result, nil
}

// Prioritize updates the priority of a queued bead
func (p *Planner) Prioritize(ctx context.Context, beadID string, priority int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, item := range p.queue {
		if item.BeadID == beadID {
			item.Priority = priority
			return nil
		}
	}

	return fmt.Errorf("bead not in queue: %s", beadID)
}

// GetQueue returns the current work queue
func (p *Planner) GetQueue(ctx context.Context) []*QueueItem {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to avoid race conditions
	queue := make([]*QueueItem, len(p.queue))
	copy(queue, p.queue)

	return queue
}

// RequeueForRework re-queues a job that failed quality check
func (p *Planner) RequeueForRework(ctx context.Context, beadID, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create new queue item with higher priority
	item := &QueueItem{
		BeadID:   beadID,
		Priority: 100, // High priority for rework
		QueuedAt: time.Now(),
		Status:   "rework",
	}

	p.queue = append(p.queue, item)

	p.events.Emit(events.EventReworkNeeded, "planner", beadID, map[string]interface{}{
		"reason": reason,
	})

	return nil
}

// processQueue processes the work queue
func (p *Planner) processQueue(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			if len(p.queue) == 0 {
				p.mu.Unlock()
				continue
			}

			available := p.stationManager.GetAvailable(ctx)
			if len(available) == 0 {
				p.mu.Unlock()
				continue
			}

			// Find highest priority queued item
			var highestPriority *QueueItem
			highestIdx := -1

			for i, item := range p.queue {
				if item.Status == "queued" {
					if highestPriority == nil || item.Priority > highestPriority.Priority {
						highestPriority = item
						highestIdx = i
					}
				}
			}

			if highestPriority != nil && len(available) > 0 {
				stationID := available[0].ID
				beadID := highestPriority.BeadID

				// Dispatch
				if err := p.Dispatch(ctx, beadID, stationID); err == nil {
					// Remove from queue
					p.queue = append(p.queue[:highestIdx], p.queue[highestIdx+1:]...)
				}
			}

			p.mu.Unlock()
		}
	}
}

// handleJobQueued handles job queued events
func (p *Planner) handleJobQueued(evt events.Event) {
	beadID, ok := evt.Payload["bead_id"].(string)
	if !ok {
		return
	}

	priority := 0
	if prio, ok := evt.Payload["priority"].(int); ok {
		priority = prio
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	item := &QueueItem{
		BeadID:   beadID,
		Priority: priority,
		QueuedAt: time.Now(),
		Status:   "queued",
	}

	p.queue = append(p.queue, item)
}

// handleStepCompleted handles step completed events
func (p *Planner) handleStepCompleted(evt events.Event) {
	stationID, _ := evt.Payload["station_id"].(string)

	// Mark station as available for new work
	p.mu.Lock()
	defer p.mu.Unlock()

	_ = p.stationManager.SetIdle(context.Background(), stationID)
}

// handleStationReady handles station ready events
func (p *Planner) handleStationReady(evt events.Event) {
	// Station is ready, trigger queue processing
	go p.processQueue(context.Background())
}

// Enqueue adds a bead to the work queue
func (p *Planner) Enqueue(ctx context.Context, beadID string, priority int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	item := &QueueItem{
		BeadID:   beadID,
		Priority: priority,
		QueuedAt: time.Now(),
		Status:   "queued",
	}

	p.queue = append(p.queue, item)

	p.events.Emit(events.EventJobQueued, "planner", beadID, map[string]interface{}{
		"priority": priority,
	})

	return nil
}

// RemoveFromQueue removes a bead from the queue
func (p *Planner) RemoveFromQueue(ctx context.Context, beadID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, item := range p.queue {
		if item.BeadID == beadID {
			p.queue = append(p.queue[:i], p.queue[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("bead not in queue: %s", beadID)
}

// GetQueueLength returns the current queue length
func (p *Planner) GetQueueLength(ctx context.Context) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.queue)
}

// GetQueuedCount returns count of queued items
func (p *Planner) GetQueuedCount(ctx context.Context) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, item := range p.queue {
		if item.Status == "queued" {
			count++
		}
	}

	return count
}

// GetActiveCount returns count of active (dispatched) items
func (p *Planner) GetActiveCount(ctx context.Context) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, item := range p.queue {
		if item.Status == "dispatched" {
			count++
		}
	}

	return count
}
