// Package beads implements the core work item system (Beads) for FactoryAI.
// Beads are the fundamental unit of work, inspired by Gas Town's metaphor.
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
	BeadAssembly  BeadType = "assembly"   // Molecule/workflow
	BeadEvent     BeadType = "event"      // System event
)

// BeadStatus represents the current state of a bead
type BeadStatus string

const (
	StatusPending    BeadStatus = "pending"
	StatusInProgress BeadStatus = "in_progress"
	StatusDone       BeadStatus = "done"
	StatusFailed     BeadStatus = "failed"
	StatusCancelled  BeadStatus = "cancelled"
)

// Bead is the core work item in FactoryAI
type Bead struct {
	ID             string                 `json:"id"`
	Type           BeadType               `json:"type"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Status         BeadStatus             `json:"status"`
	Assignee       string                 `json:"assignee,omitempty"`        // Agent ID
	ProductionLine string                 `json:"production_line,omitempty"` // Rig name
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Labels         []string               `json:"labels,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Result         string                 `json:"result,omitempty"` // Output/result of the bead
	Error          string                 `json:"error,omitempty"`  // Error message if failed
}

// BeadFilter is used to filter beads when listing
type BeadFilter struct {
	Type           BeadType
	Status         BeadStatus
	Assignee       string
	ProductionLine string
	Labels         []string
}

// BeadStore defines the interface for bead persistence
type BeadStore interface {
	Create(bead *Bead) error
	Get(id string) (*Bead, error)
	Update(bead *Bead) error
	Delete(id string) error
	List(filter BeadFilter) ([]*Bead, error)
}

// NewBead creates a new bead with defaults
func NewBead(beadType BeadType, title string) *Bead {
	now := time.Now()
	return &Bead{
		Type:      beadType,
		Title:     title,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Labels:    []string{},
		Metadata:  make(map[string]interface{}),
	}
}

// MarkInProgress updates bead status to in_progress
func (b *Bead) MarkInProgress(assignee string) {
	b.Status = StatusInProgress
	b.Assignee = assignee
	b.UpdatedAt = time.Now()
}

// MarkDone updates bead status to done
func (b *Bead) MarkDone(result string) {
	now := time.Now()
	b.Status = StatusDone
	b.Result = result
	b.UpdatedAt = now
	b.CompletedAt = &now
}

// MarkFailed updates bead status to failed
func (b *Bead) MarkFailed(err string) {
	now := time.Now()
	b.Status = StatusFailed
	b.Error = err
	b.UpdatedAt = now
	b.CompletedAt = &now
}
