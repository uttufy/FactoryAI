// Package station manages FactoryAI stations with git worktree provisioning and lifecycle management.
package station

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/store"
	"github.com/uttufy/FactoryAI/internal/tmux"
)

// StationStatus represents the status of a station
type StationStatus string

const (
	StationIdle     StationStatus = "idle"
	StationBusy     StationStatus = "busy"
	StationOffline  StationStatus = "offline"
	StationError    StationStatus = "error"
)

// Station represents a workstation with isolated worktree
type Station struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Status       StationStatus `json:"status"`
	WorktreePath string        `json:"worktree_path"`
	TmuxSession  string        `json:"tmux_session"`
	TmuxWindow   int           `json:"tmux_window"`
	TmuxPane     int           `json:"tmux_pane"`
	CurrentJob   string        `json:"current_job,omitempty"`
	OperatorID   string        `json:"operator_id,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	LastActivity time.Time     `json:"last_activity"`
}

// Manager manages stations with isolated worktrees
type Manager struct {
	projectPath string
	events      *events.EventBus
	store       *store.Store
	tmux        *tmux.Manager
	stations    map[string]*Station
	maxStations int
	mu          sync.RWMutex
}

// NewManager creates a new station manager
func NewManager(projectPath string, events *events.EventBus, store *store.Store, tmuxMgr *tmux.Manager, maxStations int) *Manager {
	return &Manager{
		projectPath: projectPath,
		events:      events,
		store:       store,
		tmux:        tmuxMgr,
		stations:    make(map[string]*Station),
		maxStations: maxStations,
	}
}

// Provision creates a new station with isolated worktree
func (m *Manager) Provision(ctx context.Context, name string) (*Station, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max stations
	if len(m.stations) >= m.maxStations {
		return nil, fmt.Errorf("max stations limit reached: %d", m.maxStations)
	}

	// Generate station ID
	stationID := fmt.Sprintf("station-%d", len(m.stations)+1)
	now := time.Now()

	// Create worktree path
	worktreePath := filepath.Join(m.projectPath, ".factory", stationID)

	// Create git worktree
	if err := m.createWorktree(worktreePath); err != nil {
		return nil, fmt.Errorf("creating worktree: %w", err)
	}

	// Create tmux session
	sessionName := fmt.Sprintf("factory-%s", stationID)
	tmuxSession, err := m.tmux.CreateSession(sessionName, worktreePath)
	if err != nil {
		// Cleanup worktree if tmux fails
		_ = m.cleanupWorktree(worktreePath)
		return nil, fmt.Errorf("creating tmux session: %w", err)
	}

	station := &Station{
		ID:           stationID,
		Name:         name,
		Status:       StationIdle,
		WorktreePath: worktreePath,
		TmuxSession:  tmuxSession.Name,
		TmuxWindow:   tmuxSession.Window,
		TmuxPane:     tmuxSession.Pane,
		CreatedAt:    now,
		LastActivity: now,
	}

	m.stations[stationID] = station

	// Emit event
	m.events.Emit(events.EventStationReady, "station_manager", stationID, map[string]interface{}{
		"name":          name,
		"worktree_path": worktreePath,
		"tmux_session":  tmuxSession.Name,
	})

	return station, nil
}

// Decommission removes a station and cleans up worktree
func (m *Manager) Decommission(ctx context.Context, stationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, ok := m.stations[stationID]
	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	// Kill tmux session
	if err := m.tmux.KillSession(station.TmuxSession); err != nil {
		return fmt.Errorf("killing tmux session: %w", err)
	}

	// Cleanup worktree
	if err := m.cleanupWorktree(station.WorktreePath); err != nil {
		return fmt.Errorf("cleaning up worktree: %w", err)
	}

	delete(m.stations, stationID)

	// Emit event
	m.events.Emit(events.EventStationOffline, "station_manager", stationID, map[string]interface{}{
		"name": station.Name,
	})

	return nil
}

// Get retrieves a station by ID
func (m *Manager) Get(ctx context.Context, stationID string) (*Station, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	station, ok := m.stations[stationID]
	if !ok {
		return nil, fmt.Errorf("station not found: %s", stationID)
	}

	return station, nil
}

// List returns all stations
func (m *Manager) List(ctx context.Context) []*Station {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var stations []*Station
	for _, station := range m.stations {
		stations = append(stations, station)
	}

	return stations
}

// GetAvailable returns stations that are idle
func (m *Manager) GetAvailable(ctx context.Context) []*Station {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var available []*Station
	for _, station := range m.stations {
		if station.Status == StationIdle {
			available = append(available, station)
		}
	}

	return available
}

// SetBusy marks a station as busy
func (m *Manager) SetBusy(ctx context.Context, stationID, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, ok := m.stations[stationID]
	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.Status = StationBusy
	station.CurrentJob = jobID
	station.LastActivity = time.Now()

	m.events.Emit(events.EventStationBusy, "station_manager", stationID, map[string]interface{}{
		"job_id": jobID,
	})

	return nil
}

// SetIdle marks a station as idle
func (m *Manager) SetIdle(ctx context.Context, stationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, ok := m.stations[stationID]
	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.Status = StationIdle
	station.CurrentJob = ""
	station.OperatorID = ""
	station.LastActivity = time.Now()

	m.events.Emit(events.EventStationReady, "station_manager", stationID, map[string]interface{}{
		"name": station.Name,
	})

	return nil
}

// createWorktree creates a git worktree for a station
func (m *Manager) createWorktree(path string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Create git worktree
	cmd := exec.Command("git", "worktree", "add", path)
	cmd.Dir = m.projectPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating worktree: %w (output: %s)", err, string(output))
	}

	return nil
}

// cleanupWorktree removes the git worktree for a station
func (m *Manager) cleanupWorktree(path string) error {
	// Remove git worktree
	cmd := exec.Command("git", "worktree", "remove", path)
	cmd.Dir = m.projectPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing worktree: %w (output: %s)", err, string(output))
	}

	// Prune worktree list
	cmd = exec.Command("git", "worktree", "prune")
	cmd.Dir = m.projectPath

	_, _ = cmd.CombinedOutput() // Ignore errors from prune

	return nil
}

// CreateWorktree creates a git worktree for a station with a specific branch
func (m *Manager) CreateWorktree(ctx context.Context, stationID, branch string) error {
	m.mu.RLock()
	station, ok := m.stations[stationID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	// Create worktree with branch
	cmd := exec.Command("git", "worktree", "add", "-b", branch, station.WorktreePath, "origin/"+branch)
	cmd.Dir = m.projectPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating worktree with branch: %w (output: %s)", err, string(output))
	}

	return nil
}

// SendCommand sends a command to a station's tmux pane
func (m *Manager) SendCommand(ctx context.Context, stationID, command string) error {
	m.mu.RLock()
	station, ok := m.stations[stationID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	return m.tmux.SendKeys(station.TmuxSession, command)
}

// CaptureOutput captures output from a station's tmux pane
func (m *Manager) CaptureOutput(ctx context.Context, stationID string) (string, error) {
	m.mu.RLock()
	station, ok := m.stations[stationID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("station not found: %s", stationID)
	}

	return m.tmux.CaptureOutput(station.TmuxSession)
}

// HealthCheck checks the health of all stations
func (m *Manager) HealthCheck(ctx context.Context) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]bool)

	for stationID, station := range m.stations {
		// Check if tmux session exists
		healthy := m.tmux.HasSession(station.TmuxSession)
		health[stationID] = healthy

		if !healthy && station.Status != StationOffline {
			// Update status to offline if session is gone
			station.Status = StationOffline
		}
	}

	return health
}

// GetStationByName retrieves a station by name
func (m *Manager) GetStationByName(ctx context.Context, name string) (*Station, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, station := range m.stations {
		if station.Name == name {
			return station, nil
		}
	}

	return nil, fmt.Errorf("station not found: %s", name)
}

// UpdateActivity updates the last activity time for a station
func (m *Manager) UpdateActivity(ctx context.Context, stationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, ok := m.stations[stationID]
	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.LastActivity = time.Now()
	return nil
}

// SetOperator sets the operator ID for a station
func (m *Manager) SetOperator(ctx context.Context, stationID, operatorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	station, ok := m.stations[stationID]
	if !ok {
		return fmt.Errorf("station not found: %s", stationID)
	}

	station.OperatorID = operatorID
	station.LastActivity = time.Now()

	return nil
}
