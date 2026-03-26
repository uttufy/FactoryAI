// Package store implements the Production Log - a SQLite-based runtime state store
// for crash recovery and system state management.
package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uttufy/FactoryAI/internal/events"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Store is the Production Log - SQLite-based runtime state
type Store struct {
	db *sql.DB
}

// NewStore creates a new Production Log store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	store := &Store{db: db}

	// Run migrations
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return store, nil
}

// Migrate runs database migrations
func (s *Store) Migrate() error {
	// Read and execute migration file
	migrationSQL, err := migrationFiles.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("reading migration file: %w", err)
	}

	if _, err := s.db.Exec(string(migrationSQL)); err != nil {
		return fmt.Errorf("executing migration: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Lease represents a lease on a resource for crash recovery
type Lease struct {
	ID           string    `json:"id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	OwnerID      string    `json:"owner_id"`
	AcquiredAt   time.Time `json:"acquired_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AcquireLease acquires a lease on a resource
func (s *Store) AcquireLease(resourceType, resourceID, ownerID string, ttl time.Duration) (*Lease, error) {
	id := fmt.Sprintf("%s:%s:%s:%d", resourceType, resourceID, ownerID, time.Now().UnixNano())
	now := time.Now()
	expiresAt := now.Add(ttl)

	lease := &Lease{
		ID:           id,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		OwnerID:      ownerID,
		AcquiredAt:   now,
		ExpiresAt:    expiresAt,
	}

	query := `
		INSERT INTO leases (id, resource_type, resource_id, owner_id, acquired_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if _, err := s.db.Exec(query, lease.ID, lease.ResourceType, lease.ResourceID, lease.OwnerID, lease.AcquiredAt, lease.ExpiresAt); err != nil {
		return nil, fmt.Errorf("inserting lease: %w", err)
	}

	return lease, nil
}

// ReleaseLease releases a lease
func (s *Store) ReleaseLease(leaseID string) error {
	query := `DELETE FROM leases WHERE id = ?`
	if _, err := s.db.Exec(query, leaseID); err != nil {
		return fmt.Errorf("deleting lease: %w", err)
	}
	return nil
}

// RenewLease renews a lease
func (s *Store) RenewLease(leaseID string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)
	query := `UPDATE leases SET expires_at = ? WHERE id = ?`
	if _, err := s.db.Exec(query, expiresAt, leaseID); err != nil {
		return fmt.Errorf("renewing lease: %w", err)
	}
	return nil
}

// GetExpiredLeases returns all expired leases
func (s *Store) GetExpiredLeases() ([]*Lease, error) {
	query := `
		SELECT id, resource_type, resource_id, owner_id, acquired_at, expires_at
		FROM leases
		WHERE expires_at < ?
		ORDER BY expires_at ASC
	`

	rows, err := s.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying expired leases: %w", err)
	}
	defer rows.Close()

	var leases []*Lease
	for rows.Next() {
		var lease Lease
		if err := rows.Scan(&lease.ID, &lease.ResourceType, &lease.ResourceID, &lease.OwnerID, &lease.AcquiredAt, &lease.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scanning lease: %w", err)
		}
		leases = append(leases, &lease)
	}

	return leases, rows.Err()
}

// LogEvent logs an event to the event log
func (s *Store) LogEvent(event events.Event) error {
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	query := `
		INSERT INTO event_log (id, type, timestamp, source, subject, payload)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	if _, err := s.db.Exec(query, event.ID, event.Type, event.Timestamp, event.Source, event.Subject, string(payloadJSON)); err != nil {
		return fmt.Errorf("inserting event log: %w", err)
	}

	return nil
}

// GetEvents retrieves events since a given time
func (s *Store) GetEvents(since time.Time, eventType events.EventType) ([]events.Event, error) {
	query := `
		SELECT id, type, timestamp, source, subject, payload
		FROM event_log
		WHERE timestamp >= ?
		AND (? = '' OR type = ?)
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, since.Unix(), eventType, eventType)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()

	var eventList []events.Event
	for rows.Next() {
		var e events.Event
		var payloadJSON string
		if err := rows.Scan(&e.ID, &e.Type, &e.Timestamp, &e.Source, &e.Subject, &payloadJSON); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}

		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &e.Payload); err != nil {
				return nil, fmt.Errorf("unmarshaling payload: %w", err)
			}
		}

		eventList = append(eventList, e)
	}

	return eventList, rows.Err()
}

// SaveDeadLetter saves a dropped event to the dead letter queue
func (s *Store) SaveDeadLetter(event events.Event, reason string) error {
	query := `
		INSERT INTO dead_letter (id, event_id, event_type, timestamp, reason)
		VALUES (?, ?, ?, ?, ?)
	`

	id := fmt.Sprintf("dl:%d", time.Now().UnixNano())
	if _, err := s.db.Exec(query, id, event.ID, event.Type, event.Timestamp, reason); err != nil {
		return fmt.Errorf("inserting dead letter: %w", err)
	}

	return nil
}

// GetDeadLetter retrieves all dead letter events
func (s *Store) GetDeadLetter() ([]events.Event, error) {
	query := `
		SELECT event_id, event_type, timestamp, reason
		FROM dead_letter
		ORDER BY timestamp DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying dead letter: %w", err)
	}
	defer rows.Close()

	var eventList []events.Event
	for rows.Next() {
		var e events.Event
		var reason string
		if err := rows.Scan(&e.ID, &e.Type, &e.Timestamp, &reason); err != nil {
			return nil, fmt.Errorf("scanning dead letter: %w", err)
		}
		eventList = append(eventList, e)
	}

	return eventList, rows.Err()
}

// ClearDeadLetter clears the dead letter queue
func (s *Store) ClearDeadLetter() error {
	query := `DELETE FROM dead_letter`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("clearing dead letter: %w", err)
	}
	return nil
}

// FactoryStatus represents the persistent factory state
type FactoryStatus struct {
	Running    bool
	PID        int64
	StartedAt  time.Time
	BootStatus string
}

// GetFactoryStatus retrieves the current factory status from database
func (s *Store) GetFactoryStatus() (*FactoryStatus, error) {
	query := `
		SELECT running, pid, started_at, boot_status
		FROM factory_status
		WHERE id = 1
	`

	var status FactoryStatus
	var running int
	var pid sql.NullInt64
	var startedAt sql.NullTime

	err := s.db.QueryRow(query).Scan(&running, &pid, &startedAt, &status.BootStatus)
	if err != nil {
		return nil, fmt.Errorf("querying factory status: %w", err)
	}

	status.Running = running == 1
	if pid.Valid {
		status.PID = pid.Int64
	}
	if startedAt.Valid {
		status.StartedAt = startedAt.Time
	}

	return &status, nil
}

// SetFactoryBooting marks the factory as booting in the database
func (s *Store) SetFactoryBooting(pid int) error {
	query := `
		UPDATE factory_status
		SET running = 0, pid = ?, started_at = ?, boot_status = 'booting'
		WHERE id = 1
	`
	if _, err := s.db.Exec(query, pid, time.Now()); err != nil {
		return fmt.Errorf("setting factory booting: %w", err)
	}
	return nil
}

// SetFactoryRunning marks the factory as running in the database
func (s *Store) SetFactoryRunning(pid int) error {
	query := `
		UPDATE factory_status
		SET running = 1, pid = ?, started_at = ?, boot_status = 'running'
		WHERE id = 1
	`
	if _, err := s.db.Exec(query, pid, time.Now()); err != nil {
		return fmt.Errorf("setting factory running: %w", err)
	}
	return nil
}

// SetFactoryStopped marks the factory as stopped in the database
func (s *Store) SetFactoryStopped() error {
	query := `
		UPDATE factory_status
		SET running = 0, pid = NULL, started_at = NULL, boot_status = 'stopped'
		WHERE id = 1
	`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("setting factory stopped: %w", err)
	}
	return nil
}

// SaveStation saves a station to the database
func (s *Store) SaveStation(station interface{}) error {
	// This is a simplified implementation
	// In a full implementation, we would properly serialize the station
	return nil
}

// GetStation retrieves a station by ID
func (s *Store) GetStation(id string) (interface{}, error) {
	query := `
		SELECT id, name, status, worktree_path, tmux_session, tmux_window, tmux_pane, current_job, operator_id, created_at, last_activity
		FROM stations
		WHERE id = ?
	`
	var st struct {
		ID           string
		Name         string
		Status       string
		WorktreePath string
		TmuxSession  string
		TmuxWindow   int
		TmuxPane     int
		CurrentJob   string
		OperatorID   string
		CreatedAt    time.Time
		LastActivity time.Time
	}
	err := s.db.QueryRow(query, id).Scan(
		&st.ID, &st.Name, &st.Status, &st.WorktreePath, &st.TmuxSession,
		&st.TmuxWindow, &st.TmuxPane, &st.CurrentJob, &st.OperatorID,
		&st.CreatedAt, &st.LastActivity,
	)
	if err != nil {
		return nil, fmt.Errorf("querying station: %w", err)
	}
	return st, nil
}

// ListStations returns all stations
func (s *Store) ListStations(ctx context.Context) ([]*Station, error) {
	query := `
		SELECT id, name, status, worktree_path, tmux_session, tmux_window, tmux_pane, current_job, operator_id, created_at, last_activity
		FROM stations
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying stations: %w", err)
	}
	defer rows.Close()

	var stations []*Station
	for rows.Next() {
		st := &Station{}
		err := rows.Scan(
			&st.ID, &st.Name, &st.Status, &st.WorktreePath, &st.TmuxSession,
			&st.TmuxWindow, &st.TmuxPane, &st.CurrentJob, &st.OperatorID,
			&st.CreatedAt, &st.LastActivity,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning station: %w", err)
		}
		stations = append(stations, st)
	}

	return stations, rows.Err()
}

// Station represents a station for listing purposes
type Station struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	WorktreePath string    `json:"worktree_path"`
	TmuxSession  string    `json:"tmux_session"`
	TmuxWindow   int       `json:"tmux_window"`
	TmuxPane     int       `json:"tmux_pane"`
	CurrentJob   string    `json:"current_job"`
	OperatorID   string    `json:"operator_id"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

// SaveOperator saves an operator to the database
func (s *Store) SaveOperator(op interface{}) error {
	// This is a simplified implementation
	return nil
}

// GetOperator retrieves an operator by ID
func (s *Store) GetOperator(id string) (interface{}, error) {
	query := `
		SELECT id, name, station_id, status, current_task, claude_session, started_at, last_heartbeat, completed_at, skills
		FROM operators
		WHERE id = ?
	`
	var op struct {
		ID            string
		Name          string
		StationID     string
		Status        string
		CurrentTask   string
		ClaudeSession string
		StartedAt     time.Time
		LastHeartbeat time.Time
		CompletedAt   *time.Time
		Skills        string
	}
	err := s.db.QueryRow(query, id).Scan(
		&op.ID, &op.Name, &op.StationID, &op.Status, &op.CurrentTask,
		&op.ClaudeSession, &op.StartedAt, &op.LastHeartbeat, &op.CompletedAt, &op.Skills,
	)
	if err != nil {
		return nil, fmt.Errorf("querying operator: %w", err)
	}
	return op, nil
}

// ListOperators returns all operators
func (s *Store) ListOperators(ctx context.Context) ([]*Operator, error) {
	query := `
		SELECT id, name, station_id, status, current_task, claude_session, started_at, last_heartbeat, completed_at, skills
		FROM operators
		ORDER BY started_at ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying operators: %w", err)
	}
	defer rows.Close()

	var operators []*Operator
	for rows.Next() {
		op := &Operator{}
		err := rows.Scan(
			&op.ID, &op.Name, &op.StationID, &op.Status, &op.CurrentTask,
			&op.ClaudeSession, &op.StartedAt, &op.LastHeartbeat, &op.CompletedAt, &op.Skills,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning operator: %w", err)
		}
		operators = append(operators, op)
	}

	return operators, rows.Err()
}

// Operator represents an operator for listing purposes
type Operator struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	StationID     string     `json:"station_id"`
	Status        string     `json:"status"`
	CurrentTask   string     `json:"current_task"`
	ClaudeSession string     `json:"claude_session"`
	StartedAt     time.Time  `json:"started_at"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
	CompletedAt   *time.Time `json:"completed_at"`
	Skills        string     `json:"skills"`
}

// UpdateHeartbeat updates the operator's last heartbeat time
func (s *Store) UpdateHeartbeat(operatorID string) error {
	query := `UPDATE operators SET last_heartbeat = ? WHERE id = ?`
	if _, err := s.db.Exec(query, time.Now(), operatorID); err != nil {
		return fmt.Errorf("updating heartbeat: %w", err)
	}
	return nil
}

// GetStuckOperators returns operators that haven't sent heartbeat recently
func (s *Store) GetStuckOperators(timeout time.Duration) ([]*Operator, error) {
	cutoff := time.Now().Add(-timeout)
	query := `
		SELECT id, name, station_id, status, current_task, claude_session, started_at, last_heartbeat, completed_at, skills
		FROM operators
		WHERE last_heartbeat < ? AND status = 'working'
		ORDER BY last_heartbeat ASC
	`

	rows, err := s.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying stuck operators: %w", err)
	}
	defer rows.Close()

	var operators []*Operator
	for rows.Next() {
		op := &Operator{}
		err := rows.Scan(
			&op.ID, &op.Name, &op.StationID, &op.Status, &op.CurrentTask,
			&op.ClaudeSession, &op.StartedAt, &op.LastHeartbeat, &op.CompletedAt, &op.Skills,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning operator: %w", err)
		}
		operators = append(operators, op)
	}

	return operators, rows.Err()
}

// IsFactoryRunning checks if the factory is currently running
// Also performs PID validation to detect stale state from crashed processes
func (s *Store) IsFactoryRunning() (bool, error) {
	status, err := s.GetFactoryStatus()
	if err != nil {
		return false, err
	}

	if !status.Running {
		return false, nil
	}

	// If PID is set, verify the process is still alive
	if status.PID > 0 {
		// Check if process with this PID exists
		// On Unix, sending signal 0 checks process existence without actually sending a signal
		process, err := os.FindProcess(int(status.PID))
		if err != nil {
			// Process doesn't exist, clean up stale state
			s.SetFactoryStopped()
			return false, nil
		}

		// Signal 0 checks if process exists (Unix only)
		if process.Signal(syscall.Signal(0)) != nil {
			// Process doesn't exist, clean up stale state
			s.SetFactoryStopped()
			return false, nil
		}
	}

	return true, nil
}
