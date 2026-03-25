// Package beads implements the core work item system (Beads) for FactoryAI.
// This file contains type definitions for Beads, Travelers, and related entities.
package beads

import (
	"time"
)

// BeadType represents the type of work item
type BeadType string

const (
	BeadTask      BeadType = "task"       // Individual task
	BeadJobTicket BeadType = "job_ticket" // Work item
	BeadWorkOrder BeadType = "work_order" // Hook attachment
	BeadBatch     BeadType = "batch"      // Convoy/bundle
	BeadSOP       BeadType = "sop"        // Standard Operating Procedure (molecule)
	BeadEvent     BeadType = "event"      // System event
	BeadRole      BeadType = "role"       // Role definition (pinned)
	BeadStation   BeadType = "station"    // Station definition (pinned)
	BeadTraveler  BeadType = "traveler"   // Station's traveler (pinned)
	BeadWorkCell  BeadType = "work_cell"  // Work cell (pinned)
)

// BeadStatus represents the current state of a bead
type BeadStatus string

const (
	StatusPending    BeadStatus = "pending"
	StatusInProgress BeadStatus = "in_progress"
	StatusBlocked    BeadStatus = "blocked"   // Waiting on dependency
	StatusDone       BeadStatus = "done"
	StatusFailed     BeadStatus = "failed"
	StatusCancelled  BeadStatus = "cancelled"
)

// BeadPersistence indicates if a bead is persistent or ephemeral
type BeadPersistence string

const (
	Persistent BeadPersistence = "persistent" // Saved to Git
	Wisp       BeadPersistence = "wisp"       // Ephemeral, burned after use
)

// Bead is the core work item in FactoryAI
type Bead struct {
	ID             string                 `json:"id"`
	Type           BeadType               `json:"type"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description,omitempty"`
	Status         BeadStatus             `json:"status"`
	Persistence    BeadPersistence        `json:"persistence"`
	Assignee       string                 `json:"assignee,omitempty"`
	StationID      string                 `json:"station_id,omitempty"`
	ParentID       string                 `json:"parent_id,omitempty"`
	Dependencies   []string               `json:"dependencies,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	CompletedAt    *string                `json:"completed_at,omitempty"`
	BurnedAt       *string                `json:"burned_at,omitempty"`
	Labels         []string               `json:"labels,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Result         string                 `json:"result,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Pinned         bool                   `json:"pinned,omitempty"`
}

// BeadFilter is used to filter beads when listing
type BeadFilter struct {
	Type        BeadType
	Status      BeadStatus
	Persistence BeadPersistence
	Assignee    string
	StationID   string
	Labels      []string
	Pinned      *bool
}

// Epic represents a hierarchical work item (parent of other beads)
type Epic struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Children []string `json:"children,omitempty"` // Child bead IDs
}

// Molecule represents a reusable workflow template (SOP)
type Molecule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Steps       []*Step   `json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
}

// Step represents a single operation in a workflow
type Step struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`     // Preferred station type
	Dependencies []string `json:"dependencies,omitempty"` // Step IDs this depends on
	Acceptance   string   `json:"acceptance,omitempty"`   // Acceptance criteria
	Gate         string   `json:"gate,omitempty"`         // Must pass before proceeding
	Timeout      int      `json:"timeout,omitempty"`      // Seconds
	MaxRetries   int      `json:"max_retries,omitempty"`
}

// Traveler represents a work order document that moves through stations
type Traveler struct {
	ID           string       `json:"id"`
	StationID    string       `json:"station_id"`
	BeadID       string       `json:"bead_id"`
	SOPID        string       `json:"sop_id,omitempty"` // Attached molecule/SOP
	Priority     int          `json:"priority"`
	Status       TravelerStatus `json:"status"`
	Deferred     bool         `json:"deferred"`
	Restart      bool         `json:"restart"`
	ReworkCount  int          `json:"rework_count"`    // Times sent back for rework
	ReworkReason string       `json:"rework_reason,omitempty"`
	AttachedAt   time.Time    `json:"attached_at"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	Result       string       `json:"result,omitempty"`
	Error        string       `json:"error,omitempty"`
}

// TravelerStatus represents the status of a traveler
type TravelerStatus string

const (
	TravelerPending  TravelerStatus = "pending"
	TravelerActive   TravelerStatus = "active"
	TravelerComplete TravelerStatus = "complete"
	TravelerFailed   TravelerStatus = "failed"
	TravelerDeferred TravelerStatus = "deferred"
	TravelerRework   TravelerStatus = "rework" // Sent back for rework
)

// Message represents mail sent between stations/agents
type Message struct {
	ID        string       `json:"id"`
	From      string       `json:"from"`
	To        string       `json:"to"`
	Subject   string       `json:"subject"`
	Body      string       `json:"body"`
	Type      MessageType  `json:"type"`
	Priority  int          `json:"priority"`
	Timestamp time.Time    `json:"timestamp"`
	Read      bool         `json:"read"`
}

// MessageType represents the type of message
type MessageType string

const (
	MsgTask     MessageType = "task"
	MsgNotify   MessageType = "notify"
	MsgEscalate MessageType = "escalate"
	MsgReply    MessageType = "reply"
	MsgSystem   MessageType = "system"
)

// AttachOption is a function that configures a traveler attachment
type AttachOption func(*Traveler)

// WithPriority sets the priority of a traveler
func WithPriority(p int) AttachOption {
	return func(t *Traveler) {
		t.Priority = p
	}
}

// WithDefer marks the traveler as deferred
func WithDefer() AttachOption {
	return func(t *Traveler) {
		t.Deferred = true
	}
}

// WithRestart marks the traveler for restart
func WithRestart() AttachOption {
	return func(t *Traveler) {
		t.Restart = true
	}
}

// NewBead creates a new bead with defaults
func NewBead(beadType BeadType, title string) *Bead {
	now := time.Now()
	return &Bead{
		Type:      beadType,
		Title:     title,
		Status:    StatusPending,
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
		Labels:    []string{},
		Metadata:  make(map[string]interface{}),
	}
}

// MarkInProgress updates bead status to in_progress
func (b *Bead) MarkInProgress(assignee string) {
	b.Status = StatusInProgress
	b.Assignee = assignee
	b.UpdatedAt = time.Now().Format(time.RFC3339)
}

// MarkDone updates bead status to done
func (b *Bead) MarkDone(result string) {
	now := time.Now()
	b.Status = StatusDone
	b.Result = result
	b.UpdatedAt = now.Format(time.RFC3339)
	formatted := now.Format(time.RFC3339)
	b.CompletedAt = &formatted
}

// MarkFailed updates bead status to failed
func (b *Bead) MarkFailed(err string) {
	now := time.Now()
	b.Status = StatusFailed
	b.Error = err
	b.UpdatedAt = now.Format(time.RFC3339)
	formatted := now.Format(time.RFC3339)
	b.CompletedAt = &formatted
}
