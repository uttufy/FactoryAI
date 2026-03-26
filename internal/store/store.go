// Package store implements the Production Log - a SQLite-based runtime state store
// for crash recovery and system state management.
package store

import (
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
