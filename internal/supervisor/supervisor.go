// Package supervisor implements the Floor Supervisor - coordinates handoffs and monitors the floor.
package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/inspector"
	"github.com/uttufy/FactoryAI/internal/operator"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/tmux"
)

// FloorStatus represents the current status of the factory floor
type FloorStatus struct {
	TotalStations   int       `json:"total_stations"`
	ActiveStations  int       `json:"active_stations"`
	IdleStations    int       `json:"idle_stations"`
	ActiveOperators int       `json:"active_operators"`
	StuckOperators  int       `json:"stuck_operators"`
	PendingMerges   int       `json:"pending_merges"`
	LastActivity    time.Time `json:"last_activity"`
}

// Supervisor oversees the factory floor
type Supervisor struct {
	events      *events.EventBus
	store       *store.Store
	inspector   *inspector.Inspector
	tmux        *tmux.Manager
}

// NewSupervisor creates a new floor supervisor
func NewSupervisor(
	events *events.EventBus,
	store *store.Store,
	inspector *inspector.Inspector,
	tmux *tmux.Manager,
) *Supervisor {
	s := &Supervisor{
		events:    events,
		store:     store,
		inspector: inspector,
		tmux:      tmux,
	}

	// Subscribe to events - commented out for now
	// events.Subscribe(events.EventOperatorHandoff, s.handleHandoff)
	// events.Subscribe(events.EventOperatorStuck, s.handleStuckOperator)
	// events.Subscribe(events.EventStepFailed, s.handleStepFailed)

	return s
}

// CoordinateHandoff coordinates graceful handoff between operators
func (s *Supervisor) CoordinateHandoff(ctx context.Context, fromStationID, toStationID string, operatorPool *operator.Pool) error {
	// Get current operator at from station
	// This would need operatorPool to be passed or stored
	// For now, emit event and let handler process
	s.events.Emit(events.EventOperatorHandoff, "supervisor", fromStationID, map[string]interface{}{
		"from_station": fromStationID,
		"to_station":   toStationID,
	})

	return nil
}

// Start begins supervision
func (s *Supervisor) Start(ctx context.Context) error {
	// Start periodic health checks
	go s.healthCheckLoop(ctx)

	return nil
}

// GetStatus returns current floor status
func (s *Supervisor) GetStatus(ctx context.Context, stationManager *station.Manager, operatorPool *operator.Pool) (*FloorStatus, error) {
	stations := stationManager.List(ctx)

	status := &FloorStatus{
		TotalStations: len(stations),
		LastActivity:  time.Now(),
	}

	for _, st := range stations {
		switch st.Status {
		case station.StationBusy:
			status.ActiveStations++
		case station.StationIdle:
			status.IdleStations++
		}
	}

	status.ActiveOperators = operatorPool.GetActiveCount(ctx)

	// Get stuck operators
	stuck := operatorPool.GetStuck(ctx, 5*time.Minute)
	status.StuckOperators = len(stuck)

	return status, nil
}

// handleHandoff handles operator handoff events
func (s *Supervisor) handleHandoff(evt events.Event) {
	fromStation, ok := evt.Payload["from_station"].(string)
	if !ok {
		return
	}

	toStation, ok := evt.Payload["to_station"].(string)
	if !ok {
		return
	}

	// Log the handoff
	_ = fmt.Sprintf("Handoff from %s to %s", fromStation, toStation)
}

// handleStuckOperator handles stuck operator events
func (s *Supervisor) handleStuckOperator(evt events.Event) {
	operatorID, ok := evt.Payload["operator_id"].(string)
	if !ok {
		return
	}

	// Log stuck operator
	_ = fmt.Sprintf("Operator %s is stuck", operatorID)

	// Could trigger automatic recovery here
	// For now, just log
}

// handleStepFailed handles step failed events
func (s *Supervisor) handleStepFailed(evt events.Event) {
	stationID, ok := evt.Payload["station_id"].(string)
	if !ok {
		return
	}

	reason, ok := evt.Payload["reason"].(string)
	if !ok {
		reason = "unknown reason"
	}

	// Log failure
	_ = fmt.Sprintf("Step failed at station %s: %s", stationID, reason)
}

// healthCheckLoop runs periodic health checks
func (s *Supervisor) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runHealthCheck(ctx)
		}
	}
}

// runHealthCheck performs a health check on the floor
func (s *Supervisor) runHealthCheck(ctx context.Context) {
	// Emit health check event
	s.events.Emit(events.EventHealthOK, "supervisor", "floor", map[string]interface{}{
		"timestamp": time.Now().Unix(),
	})
}

// MonitorOperators monitors operators for stuck states
func (s *Supervisor) MonitorOperators(ctx context.Context, operatorPool *operator.Pool) error {
	stuck := operatorPool.GetStuck(ctx, 5*time.Minute)

	for _, op := range stuck {
		s.events.Emit(events.EventOperatorStuck, "supervisor", op.ID, map[string]interface{}{
			"station_id":     op.StationID,
			"last_heartbeat": op.LastHeartbeat,
		})
	}

	return nil
}

// VerifyStations verifies all stations are healthy
func (s *Supervisor) VerifyStations(ctx context.Context, stationManager *station.Manager) error {
	health := stationManager.HealthCheck(ctx)

	for stationID, isHealthy := range health {
		if !isHealthy {
			s.events.Emit(events.EventStationOffline, "supervisor", stationID, map[string]interface{}{
				"reason": "health check failed",
			})
		}
	}

	return nil
}

// Escalate escalates an issue to human attention
func (s *Supervisor) Escalate(ctx context.Context, issue string) error {
	s.events.Emit(events.EventQualityFailed, "supervisor", "escalation", map[string]interface{}{
		"issue":     issue,
		"timestamp": time.Now().Unix(),
	})

	return nil
}

// RequestNudge requests a nudge be sent to an operator
func (s *Supervisor) RequestNudge(ctx context.Context, operatorID, message string) error {
	s.events.Emit(events.EventOperatorStuck, "supervisor", operatorID, map[string]interface{}{
		"nudge_message": message,
	})

	return nil
}

// GetStationStatus gets the status of a specific station
func (s *Supervisor) GetStationStatus(ctx context.Context, stationID string, stationManager *station.Manager) (*station.Station, error) {
	return stationManager.Get(ctx, stationID)
}

// GetOperatorStatus gets the status of operators at a station
func (s *Supervisor) GetOperatorStatus(ctx context.Context, stationID string, operatorPool *operator.Pool) (*operator.Operator, error) {
	return operatorPool.GetOperatorByStation(ctx, stationID)
}

// ScheduleHealthCheck schedules a one-time health check
func (s *Supervisor) ScheduleHealthCheck(ctx context.Context, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		s.runHealthCheck(ctx)
	}()
}
