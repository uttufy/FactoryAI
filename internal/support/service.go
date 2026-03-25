// Package support implements the Support Service - combined maintenance, reliability, and expeditor.
package support

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/operator"
	"github.com/uttufy/FactoryAI/internal/station"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/tmux"
)

// TaskType represents the type of support task
type TaskType string

const (
	TaskCleanup     TaskType = "cleanup"
	TaskHealthCheck TaskType = "health_check"
	TaskNudge       TaskType = "nudge"
	TaskRecovery    TaskType = "recovery"
)

// Task represents a support task
type Task struct {
	ID          string       `json:"id"`
	Type        TaskType     `json:"type"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// HealthReport represents a system health report
type HealthReport struct {
	DatabaseOK     bool     `json:"database_ok"`
	TmuxOK         bool     `json:"tmux_ok"`
	BeadsOK        bool     `json:"beads_ok"`
	DiskSpaceMB    int64    `json:"disk_space_mb"`
	ActiveStations int      `json:"active_stations"`
	ExpiredLeases  int      `json:"expired_leases"`
	Errors         []string `json:"errors,omitempty"`
}

// Service is the combined support service (maintenance + reliability + expeditor)
type Service struct {
	events  *events.EventBus
	store   *store.Store
	tmux    *tmux.Manager
	tasks   map[string]*Task
}

// NewService creates a new support service
func NewService(events *events.EventBus, store *store.Store, tmux *tmux.Manager) *Service {
	s := &Service{
		events: events,
		store:  store,
		tmux:   tmux,
		tasks:  make(map[string]*Task),
	}

	// Subscribe to all events (read-only observer) - commented out for now
	// events.SubscribeAll(func(evt events.Event) {
	// 	s.handleEvent(evt)
	// })

	return s
}

// Start begins the support service
func (s *Service) Start(ctx context.Context) error {
	// Start periodic tasks
	go s.periodicHealthCheck(ctx)
	go s.periodicCleanup(ctx)
	go s.leaseRecoveryLoop(ctx)

	return nil
}

// RunCleanup cleans up completed stations and old data
func (s *Service) RunCleanup(ctx context.Context) error {
	task := &Task{
		ID:          fmt.Sprintf("cleanup-%d", time.Now().Unix()),
		Type:        TaskCleanup,
		Description: "Clean up completed stations and old data",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	s.tasks[task.ID] = task

	// Perform cleanup
	// 1. Clean up dead letter queue
	if err := s.store.ClearDeadLetter(); err != nil {
		task.Status = "failed"
		return fmt.Errorf("clearing dead letter: %w", err)
	}

	// 2. Check for expired leases
	expired, err := s.store.GetExpiredLeases()
	if err != nil {
		task.Status = "failed"
		return fmt.Errorf("getting expired leases: %w", err)
	}

	// 3. Clean up old events (older than 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)
	oldEvents, err := s.store.GetEvents(cutoff, "")
	if err == nil {
		// Would need a delete method in store
		_ = oldEvents
	}

	now := time.Now()
	task.Status = "completed"
	task.CompletedAt = &now

	s.events.Emit(events.EventCleanupDone, "support_service", task.ID, map[string]interface{}{
		"expired_leases": len(expired),
	})

	return nil
}

// RunHealthCheck checks system health
func (s *Service) RunHealthCheck(ctx context.Context) (*HealthReport, error) {
	report := &HealthReport{
		Errors: make([]string, 0),
	}

	// Check database
	report.DatabaseOK = s.store != nil
	if !report.DatabaseOK {
		report.Errors = append(report.Errors, "database not initialized")
	}

	// Check tmux
	if s.tmux != nil {
		sessions, err := s.tmux.ListSessions()
		report.TmuxOK = err == nil
		report.ActiveStations = len(sessions)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("tmux error: %v", err))
		}
	}

	// Check disk space
	if stat, err := os.Stat("."); err == nil {
		report.DiskSpaceMB = stat.Size() / (1024 * 1024)
	}

	// Check expired leases
	expired, err := s.store.GetExpiredLeases()
	if err == nil {
		report.ExpiredLeases = len(expired)
	}

	// Emit health status
	if len(report.Errors) == 0 {
		s.events.Emit(events.EventHealthOK, "support_service", "health_check", map[string]interface{}{
			"report": report,
		})
	}

	return report, nil
}

// Nudge sends a nudge to an operator
func (s *Service) Nudge(ctx context.Context, operatorID, message string) error {
	task := &Task{
		ID:          fmt.Sprintf("nudge-%s-%d", operatorID, time.Now().Unix()),
		Type:        TaskNudge,
		Description: fmt.Sprintf("Nudge operator %s: %s", operatorID, message),
		Status:      "running",
		StartedAt:   time.Now(),
	}
	s.tasks[task.ID] = task

	// Emit nudge event
	s.events.Emit(events.EventOperatorStuck, "support_service", operatorID, map[string]interface{}{
		"nudge_message": message,
		"task_id":       task.ID,
	})

	now := time.Now()
	task.Status = "completed"
	task.CompletedAt = &now

	return nil
}

// NudgeAll sends nudge to all operators
func (s *Service) NudgeAll(ctx context.Context, message string) error {
	task := &Task{
		ID:          fmt.Sprintf("nudge-all-%d", time.Now().Unix()),
		Type:        TaskNudge,
		Description: fmt.Sprintf("Nudge all operators: %s", message),
		Status:      "running",
		StartedAt:   time.Now(),
	}
	s.tasks[task.ID] = task

	// Emit broadcast nudge event
	s.events.Emit(events.EventOperatorStuck, "support_service", "all", map[string]interface{}{
		"nudge_message": message,
		"task_id":       task.ID,
		"broadcast":     true,
	})

	now := time.Now()
	task.Status = "completed"
	task.CompletedAt = &now

	return nil
}

// RecoverExpiredLeases recovers work from expired leases
func (s *Service) RecoverExpiredLeases(ctx context.Context) error {
	expired, err := s.store.GetExpiredLeases()
	if err != nil {
		return fmt.Errorf("getting expired leases: %w", err)
	}

	for _, lease := range expired {
		// Log the expired lease
		_ = fmt.Sprintf("Expired lease: %s (type: %s, resource: %s)", lease.ID, lease.ResourceType, lease.ResourceID)

		// Emit recovery event
		s.events.Emit(events.EventOperatorStuck, "support_service", lease.ResourceID, map[string]interface{}{
			"lease_id":       lease.ID,
			"resource_type":  lease.ResourceType,
			"owner_id":       lease.OwnerID,
			"recovery_action": "lease_expired",
		})
	}

	return nil
}

// handleEvent handles all events (read-only observer)
func (s *Service) handleEvent(evt events.Event) {
	// Log events for monitoring
	// Could store in database for analysis
	switch evt.Type {
	case events.EventOperatorStuck:
		// Track stuck operators
		s.tasks[evt.ID] = &Task{
			ID:          evt.ID,
			Type:        TaskRecovery,
			Description: "Operator stuck - recovery needed",
			Status:      "pending",
			StartedAt:   time.Now(),
		}
	case events.EventQualityFailed:
		// Track quality failures
		s.tasks[evt.ID] = &Task{
			ID:          evt.ID,
			Type:        TaskRecovery,
			Description: "Quality failure - rework needed",
			Status:      "pending",
			StartedAt:   time.Now(),
		}
	}
}

// periodicHealthCheck runs periodic health checks
func (s *Service) periodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.RunHealthCheck(ctx)
		}
	}
}

// periodicCleanup runs periodic cleanup
func (s *Service) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.RunCleanup(ctx)
		}
	}
}

// leaseRecoveryLoop runs periodic lease recovery
func (s *Service) leaseRecoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.RecoverExpiredLeases(ctx)
		}
	}
}

// GetTasks returns all tasks
func (s *Service) GetTasks() []*Task {
	var tasks []*Task
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetTask returns a specific task
func (s *Service) GetTask(id string) (*Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return task, nil
}

// CleanupStation cleans up a dead station
func (s *Service) CleanupStation(ctx context.Context, stationID string, stationManager *station.Manager) error {
	// Decommission the station
	if err := stationManager.Decommission(ctx, stationID); err != nil {
		return fmt.Errorf("decommissioning station: %w", err)
	}

	s.events.Emit(events.EventCleanupDone, "support_service", stationID, map[string]interface{}{
		"action": "station_decommissioned",
	})

	return nil
}

// CleanupDeadOperator cleans up a dead operator
func (s *Service) CleanupDeadOperator(ctx context.Context, operatorID string, operatorPool *operator.Pool) error {
	// Mark operator as failed
	if err := operatorPool.MarkFailed(ctx, operatorID, "operator died"); err != nil {
		return fmt.Errorf("marking operator failed: %w", err)
	}

	s.events.Emit(events.EventCleanupDone, "support_service", operatorID, map[string]interface{}{
		"action": "operator_cleaned_up",
	})

	return nil
}

// MonitorStations monitors stations for health issues
func (s *Service) MonitorStations(ctx context.Context, stationManager *station.Manager) error {
	health := stationManager.HealthCheck(ctx)

	for stationID, isHealthy := range health {
		if !isHealthy {
			s.events.Emit(events.EventStationOffline, "support_service", stationID, map[string]interface{}{
				"reason": "health_check_failed",
			})
		}
	}

	return nil
}
