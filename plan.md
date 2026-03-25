# FactoryAI - Low-Level Design (LLD)

## Overview

FactoryAI is a multi-agent workspace manager that orchestrates parallel AI agents working on software development tasks. It uses real manufacturing factory concepts and terminology, adapted for software production.

**Key Integrations:**
- **Beads CLI**: Uses the `beads` CLI tool (github.com/steveyegge/beads) for work item management
- **tmux**: Primary UI and session management
- **Claude Code**: The underlying AI agent (`claude` binary)
- **SQLite**: Runtime state storage for crash recovery

---

## Architectural Principles

### Source of Truth Hierarchy

FactoryAI uses a strict hierarchy to avoid consistency issues between data stores:

```
┌─────────────────────────────────────────────────────────┐
│ DATA PLANE AUTHORITY HIERARCHY                          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ BEADS (Git-backed) — MASTER SOURCE OF TRUTH     │   │
│  │ ════════════════════════════════════════════    │   │
│  │ Authority: Job definitions, SOPs, final status  │   │
│  │ Mutation: Only through beads CLI                │   │
│  │ Recovery: Git is the restore point              │   │
│  └─────────────────────────────────────────────────┘   │
│                         │                              │
│                         │ Beads is source              │
│                         ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │ PRODUCTION LOG (SQLite) — RUNTIME CACHE         │   │
│  │ ════════════════════════════════════════════    │   │
│  │ Authority: NOTHING (derived state only)         │   │
│  │ Contains: Leases, heartbeats, transient state   │   │
│  │ Rule: If conflict, BEADS WINS                   │   │
│  │ Recovery: Can be rebuilt from BEADS + events    │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Recovery Rule**: On startup, if SQLite says "Job 123 in progress" but Beads says "Job 123 done" → Trust Beads, clear SQLite entry.

### Control Room Authority

The Control Room has a clear command hierarchy to avoid conflicting decisions:

```
CONTROL ROOM - SINGLE POINT OF AUTHORITY
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ PLANT DIRECTOR — SINGLE AUTHORITY               │   │
│  │ ════════════════════════════════════════════    │   │
│  │ Owns: Schedule decisions, user interface        │   │
│  │ Can: Override planner, pause factory, escalate  │   │
│  │ Cannot: Execute work directly                   │   │
│  └───────────────────────┬─────────────────────────┘   │
│                          │ Commands                    │
│          ┌───────────────┼───────────────┐             │
│          ▼               ▼               ▼             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐   │
│  │ PLANNER      │ │ ENGINEER     │ │ (no others)  │   │
│  │ ════════════ │ │ ════════════ │ │              │   │
│  │ Executes:    │ │ Executes:    │ │              │   │
│  │ - Dispatch   │ │ - DAG eval   │ │              │   │
│  │ - Queue mgmt │ │ - Routing    │ │              │   │
│  │              │ │              │ │              │   │
│  │ CANNOT:      │ │ CANNOT:      │ │              │   │
│  │ - Override   │ │ - Dispatch   │ │              │   │
│  │ - Escalate   │ │ - Override   │ │              │   │
│  └──────────────┘ └──────────────┘ └──────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Rule**: Only Plant Director can make policy decisions. Planner and Engineer are executors with no override authority.

### Event Bus Specification

| Property | Value | Rationale |
|----------|-------|-----------|
| **Durability** | In-memory only | Simplicity for v1 |
| **Replay** | Yes, via SQLite log | Events logged but not persisted before delivery |
| **Ordering** | Per-event-type FIFO | Within same event type, order preserved |
| **Delivery** | At-most-once | No redelivery if handler crashes |
| **Backpressure** | Drop + log | If buffer full, drop and log to dead_letter |
| **Blocking** | Non-blocking publish | Publishers never block |

**Subscription Matrix:**

| Event | Director | Planner | Engineer | Supervisor | Support | Assembly |
|-------|----------|---------|----------|------------|---------|----------|
| JobQueued | - | Subscribe | - | - | - | - |
| StationReady | - | Subscribe | - | - | - | - |
| StepQueued | - | - | Subscribe | - | - | - |
| StepDone | Subscribe | - | Subscribe | - | - | Subscribe |
| StepFailed | Subscribe | - | Subscribe | Subscribe | - | - |
| OperatorStuck | - | - | - | Subscribe | Subscribe | - |
| MergeReady | - | - | - | - | - | Subscribe |

### Support Service Boundaries

Support Service is a READ-ONLY observer with limited actions:

```
SUPPORT SERVICE BOUNDARIES
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  CAN DO:                                                │
│  ✓ Subscribe to ALL events (read-only)                 │
│  ✓ Emit: Stuck, HealthOK, CleanupDone                  │
│  ✓ Nudge operators (via Expeditor)                     │
│  ✓ Clean up dead stations                              │
│                                                         │
│  CANNOT DO:                                             │
│  ✗ Dispatch work                                       │
│  ✗ Modify job status                                   │
│  ✗ Override planner decisions                          │
│  ✗ Create/modify travelers                             │
│                                                         │
│  ENFORCEMENT:                                           │
│  - No access to TravelerManager                        │
│  - No write access to Beads                            │
│  - Can only write to: leases, event_log, dead_letter   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Station Execution Lifecycle

```
STATION EXECUTION LIFECYCLE
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  1. CLAIMING (Lease Acquisition)                        │
│     ┌─────────────────────────────────────────────┐    │
│     │ Station requests lease on Bead              │    │
│     │ SQLite: INSERT INTO leases (...)            │    │
│     │ If conflict (lease exists) → Wait or fail   │    │
│     └─────────────────────────────────────────────┘    │
│                                                         │
│  2. EXECUTION (with Heartbeat)                          │
│     ┌─────────────────────────────────────────────┐    │
│     │ Operator sends heartbeat every N seconds    │    │
│     │ SQLite: UPDATE leases SET expires_at=...    │    │
│     │ If heartbeat missed > TTL → Lease expires   │    │
│     └─────────────────────────────────────────────┘    │
│                                                         │
│  3. COMPLETION (Lease Release)                          │
│     ┌─────────────────────────────────────────────┐    │
│     │ On success: Release lease, update Beads     │    │
│     │ On failure: Release lease, mark Bead failed │    │
│     │ On crash: Lease auto-expires, reclaimable   │    │
│     └─────────────────────────────────────────────┘    │
│                                                         │
│  4. RECOVERY (Crash Recovery)                           │
│     ┌─────────────────────────────────────────────┐    │
│     │ On startup: Find expired leases             │    │
│     │ Check Beads for actual status               │    │
│     │ If Beads says "in_progress" → Re-queue      │    │
│     │ If Beads says "done" → Clear lease          │    │
│     └─────────────────────────────────────────────┘    │
│                                                         │
│  5. IDEMPOTENCY                                         │
│     ┌─────────────────────────────────────────────┐    │
│     │ All operations safe to retry                │    │
│     │ Step completion checks current state first  │    │
│     │ If already done → Skip (no-op)              │    │
│     └─────────────────────────────────────────────┘    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Rework Loops

Quality failures trigger rework:

```
PRODUCTION FLOOR WITH REWORK
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  ┌─────────┐    ┌─────────────┐    ┌───────────────┐   │
│  │ Station │───▶│ Quality     │───▶│ Final Assembly│   │
│  │         │    │ Inspector   │    │               │   │
│  └─────────┘    └──────┬──────┘    └───────────────┘   │
│       ▲                │                                │
│       │         FAILED │                                │
│       └────────────────┘                                │
│              REWORK LOOP                                │
│                                                         │
│  Events:                                                │
│  - QualityFailed → Planner re-queues job               │
│  - ReworkNeeded → Station receives job with context    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## MVP v1 Architecture (Collapsed Roles)

For v1, roles are collapsed to reduce complexity:

| Full Architecture | MVP v1 |
|-------------------|--------|
| Plant Director | **Plant Director** (includes planning + process engineering) |
| Production Planner | ↳ merged into Director |
| Process Engineer | ↳ merged into Director |
| Floor Supervisor | **Floor Supervisor** (includes inspection) |
| Quality Inspector | ↳ merged into Supervisor |
| Maintenance Crew | **Support Service** (single background daemon) |
| Reliability Engineer | ↳ merged into Support Service |
| Expeditor | ↳ merged into Support Service |

**MVP v1 Roles (3 total):**
1. **Plant Director** - User interface, planning, dispatching, DAG evaluation
2. **Floor Supervisor** - Monitoring, inspection, handoffs
3. **Support Service** - Heartbeats, stuck detection, cleanup, nudges

---

## Manufacturing Terminology

### Control Room (Management)

| Manufacturing Role | FactoryAI Role | Responsibility |
|-------------------|----------------|----------------|
| **Plant Director** | Plant Director | Overall factory operations, receives user requests, single authority |
| **Production Planner** | Production Planner | Schedules work, manages capacity, dispatches to stations |
| **Process Engineer** | Process Engineer | Designs workflows (DAG), defines routing between stations |

### Production Floor (Execution)

| Manufacturing Role | FactoryAI Role | Responsibility |
|-------------------|----------------|----------------|
| **Station** | Station | A single workspace (git worktree + tmux pane) |
| **Operator** | Operator | AI agent working at a station |
| **Work Cell** | Work Cell | Group of parallel stations attacking work together |
| **Traveler** | Traveler | Work order document that moves through stations |
| **Floor Supervisor** | Floor Supervisor | Oversees floor, coordinates handoffs |
| **Quality Inspector** | Quality Inspector | Verifies work, helps stuck operators |
| **Final Assembly** | Final Assembly | Merges completed work into main branch |

### Support Services

| Manufacturing Role | FactoryAI Role | Responsibility |
|-------------------|----------------|----------------|
| **Maintenance Crew** | Maintenance Crew | Keeps factory running |
| **Reliability Engineer** | Reliability Engineer | Watchdog, ensures uptime |
| **Expeditor** | Expeditor | Rushes urgent work, nudges operators |

### Material Flow (Data)

| Manufacturing Concept | FactoryAI Concept |
|----------------------|-------------------|
| **Job / Task** | Bead (work item) |
| **Production Batch** | Batch (tracked unit of delivery) |
| **Standard Operating Procedure (SOP)** | Molecule (workflow) |
| **SOP Template** | Formula (TOML recipe) |
| **Andon Board** | Event Bus (signaling system) |
| **Production Log** | Runtime State (SQLite) |
| **WIP Inventory** | Work In Progress tracking |

---

## Four-Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│ LAYER 1: INTERFACE & OBSERVABILITY                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ CLI (Cobra)  │  │ TUI (Bubble) │  │ HTTP Server (opt-in)     │  │
│  │ - Commands   │  │ - Dashboard  │  │ - /health, /metrics      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│ LAYER 2: CONTROL ROOM (The Brain)                                   │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    ANDON BOARD (Event Bus)                    │  │
│  │  Events: JobQueued, StationReady, StepDone, MergeReady        │  │
│  │  Subscribers: Planner, Engineer, Supervisor, Support, Assembly│  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │ PLANT DIRECTOR (Single Authority)                           │     │
│  │ - User interface, strategy, final decisions                 │     │
│  │ - Batch management, escalation                              │     │
│  │ - Delegates to Planner & Engineer                           │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌────────────────────┐  ┌────────────────────────────────────┐     │
│  │ PRODUCTION PLANNER │  │ PROCESS ENGINEER                   │     │
│  │ - Single dispatcher│  │ - DAG workflow evaluation          │     │
│  │ - Assigns travelers│  │ - Queues ready steps in parallel  │     │
│  │ - No race conditions│  │ - Supports linear as trivial DAG │     │
│  └────────────────────┘  └────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│ LAYER 3: PRODUCTION FLOOR (Factory Floor)                           │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ STATION MANAGER                                                │ │
│  │ - Git worktree provisioning, isolation, lifecycle management   │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────┐ │
│  │ STATION 1      │  │ STATION 2      │  │ STATION N              │ │
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐         │ │
│  │ │ Operator   │ │  │ │ Operator   │ │  │ │ Operator   │         │ │
│  │ │ (AI Agent) │ │  │ │ (AI Agent) │ │  │ │ (AI Agent) │         │ │
│  │ └────────────┘ │  │ └────────────┘ │  │ └────────────┘         │ │
│  │ Traveler: #123 │  │ Traveler: #124 │  │ Traveler: #125         │ │
│  │ Worktree: ./.1 │  │ Worktree: ./.2 │  │ Worktree: ./.N         │ │
│  └────────────────┘  └────────────────┘  └────────────────────────┘ │
│                                                                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────────────┐ │
│  │ FLOOR          │  │ QUALITY        │  │ FINAL ASSEMBLY         │ │
│  │ SUPERVISOR     │  │ INSPECTOR      │  │ - Parallel merge queue │ │
│  │ - Coordinates  │  │ - Verifies     │  │ - Conflict detection   │ │
│  │ - Handoffs     │◀─│ - Rework loops │  │ - Intelligent merging  │ │
│  └────────────────┘  └────────────────┘  └────────────────────────┘ │
│         ▲                                                    ▲       │
│         │            REWORK LOOP                             │       │
│         └────────────────────────────────────────────────────┘       │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ SUPPORT SERVICE (Maintenance + Reliability + Expeditor)        │ │
│  │ - Subscribes to ALL events (read-only)                         │ │
│  │ - Emits: Stuck, HealthOK, CleanupDone                          │ │
│  │ - Actions: Nudge operators, cleanup dead stations              │ │
│  │ - CANNOT: Dispatch, modify job status, override planner        │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│ LAYER 4: DATA PLANE (The Ledger)                                    │
│  ┌────────────────────────────┐  ┌────────────────────────────────┐ │
│  │ BEADS (Jobs/SOPs)          │  │ PRODUCTION LOG (SQLite)        │ │
│  │ - MASTER SOURCE OF TRUTH   │  │ - RUNTIME CACHE ONLY           │ │
│  │ - Business logic state     │  │ - Station assignments          │ │
│  │ - Git-backed durability    │  │ - Operator heartbeats & leases │ │
│  │ - Work items, templates    │  │ - Step execution state         │ │
│  │                            │  │ - Crash recovery data          │ │
│  └────────────────────────────┘  └────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Event Bus (Andon Board)

The central nervous system for reactive communication. Replaces polling-based patrol loops.

```go
// internal/events/bus.go
package events

import (
    "sync"
    "context"
    "time"
)

// Event types that flow through the Andon Board
type EventType string

const (
    // Job lifecycle
    EventJobCreated       EventType = "job.created"
    EventJobQueued        EventType = "job.queued"
    EventJobStarted       EventType = "job.started"
    EventJobCompleted     EventType = "job.completed"
    EventJobFailed        EventType = "job.failed"
    
    // Station lifecycle
    EventStationReady     EventType = "station.ready"
    EventStationBusy      EventType = "station.busy"
    EventStationOffline   EventType = "station.offline"
    
    // Step execution (DAG)
    EventStepQueued       EventType = "step.queued"
    EventStepStarted      EventType = "step.started"
    EventStepCompleted    EventType = "step.completed"
    EventStepFailed       EventType = "step.failed"
    EventDependenciesMet  EventType = "step.dependencies_met"
    
    // Merge operations
    EventMergeReady       EventType = "merge.ready"
    EventMergeStarted     EventType = "merge.started"
    EventMergeCompleted   EventType = "merge.completed"
    EventMergeConflict    EventType = "merge.conflict"
    
    // Operator lifecycle
    EventOperatorSpawned  EventType = "operator.spawned"
    EventOperatorIdle     EventType = "operator.idle"
    EventOperatorStuck    EventType = "operator.stuck"
    EventOperatorHandoff  EventType = "operator.handoff"
    
    // Quality & Rework
    EventQualityFailed    EventType = "quality.failed"
    EventReworkNeeded     EventType = "rework.needed"
    
    // System events
    EventHeartbeat        EventType = "system.heartbeat"
    EventFactoryShutdown  EventType = "system.shutdown"
    EventHealthOK         EventType = "system.health_ok"
    EventCleanupDone      EventType = "system.cleanup_done"
)

type Event struct {
    ID        string                 `json:"id"`
    Type      EventType              `json:"type"`
    Timestamp int64                  `json:"timestamp"`
    Source    string                 `json:"source"`    // Who emitted this
    Subject   string                 `json:"subject"`   // What this is about
    Payload   map[string]interface{} `json:"payload,omitempty"`
}

type Handler func(Event)

// EventBus is the Andon Board - in-memory pub/sub
type EventBus struct {
    mu          sync.RWMutex
    handlers    map[EventType][]Handler
    all         []Handler  // Handlers that want all events
    buffer      int        // Channel buffer size
    deadLetter  []Event    // Dropped events
    eventLog    *store.Store
}

func NewEventBus(bufferSize int, store *store.Store) *EventBus

// Subscribe registers a handler for specific event types
// Returns an unsubscribe function
func (b *EventBus) Subscribe(eventType EventType, handler Handler) func()

// SubscribeAll registers a handler for all events (for logging/metrics)
func (b *EventBus) SubscribeAll(handler Handler) func()

// Publish emits an event to all subscribers (non-blocking)
func (b *EventBus) Publish(event Event)

// PublishAsync emits an event without blocking, logs to dead letter if buffer full
func (b *EventBus) PublishAsync(event Event)

// WaitFor blocks until an event of the given type is received
func (b *EventBus) WaitFor(ctx context.Context, eventType EventType, timeout time.Duration) (Event, error)

// GetDeadLetter returns events that were dropped
func (b *EventBus) GetDeadLetter() []Event
```

### 2. Beads Integration

FactoryAI integrates with the `beads` CLI tool for all work item management.

```go
// internal/beads/client.go
package beads

import (
    "os/exec"
    "encoding/json"
)

// Client wraps the beads CLI
type Client struct {
    binaryPath string
    workingDir string
}

func NewClient(binaryPath, workingDir string) (*Client, error)

// Execute runs a beads command and returns the output
func (c *Client) Execute(args ...string) (string, error)

// Bead operations
func (c *Client) Create(beadType, title string) (*Bead, error)
func (c *Client) Get(id string) (*Bead, error)
func (c *Client) Update(id string, updates map[string]interface{}) error
func (c *Client) List(filter BeadFilter) ([]*Bead, error)
func (c *Client) Delete(id string) error
func (c *Client) Close(id string) error
func (c *Client) Ready() ([]*Bead, error)  // Get ready work

// Wisp operations (ephemeral beads)
func (c *Client) CreateWisp(beadType, title string) (*Bead, error)
func (c *Client) Burn(id string) error
func (c *Client) Squash(id, summary string) error

// Epic operations (hierarchical work)
func (c *Client) CreateEpic(title string) (*Epic, error)
func (c *Client) AddChild(epicID, childID string) error
func (c *Client) GetChildren(epicID string) ([]*Bead, error)

// Molecule operations
func (c *Client) CreateMolecule(name string, steps []*Step) (*Molecule, error)
func (c *Client) InstantiateMolecule(protoID string, vars map[string]string) (*Molecule, error)

// Traveler operations (was "hooks")
func (c *Client) AttachTraveler(stationID, beadID string, opts ...AttachOption) error
func (c *Client) GetTraveler(stationID string) (*Traveler, error)
func (c *Client) ClearTraveler(stationID string) error

// Mail operations
func (c *Client) SendMail(from, to, subject, body string) error
func (c *Client) ReadMail(stationID string) ([]*Message, error)

// Cross-rig routing
func (c *Client) Route(prefix string) (*Client, error)
```

### 3. Bead Types

```go
// internal/beads/types.go
package beads

type BeadType string

const (
    BeadTask      BeadType = "task"       // Individual task
    BeadJobTicket BeadType = "job_ticket" // Work item
    BeadBatch     BeadType = "batch"      // Production batch
    BeadSOP       BeadType = "sop"        // Standard Operating Procedure (molecule)
    BeadEvent     BeadType = "event"      // System event
    BeadRole      BeadType = "role"       // Role definition (pinned)
    BeadStation   BeadType = "station"    // Station definition (pinned)
    BeadTraveler  BeadType = "traveler"   // Station's traveler (pinned)
    BeadWorkCell  BeadType = "work_cell"  // Work cell (pinned)
)

type BeadStatus string

const (
    StatusPending    BeadStatus = "pending"
    StatusInProgress BeadStatus = "in_progress"
    StatusBlocked    BeadStatus = "blocked"   // Waiting on dependency
    StatusDone       BeadStatus = "done"
    StatusFailed     BeadStatus = "failed"
    StatusCancelled  BeadStatus = "cancelled"
)

type BeadPersistence string

const (
    Persistent BeadPersistence = "persistent" // Saved to Git
    Wisp       BeadPersistence = "wisp"       // Ephemeral, burned after use
)

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

type BeadFilter struct {
    Type        BeadType
    Status      BeadStatus
    Persistence BeadPersistence
    Assignee    string
    StationID   string
    Labels      []string
    Pinned      *bool
}
```

### 4. Travelers (Work Orders)

A Traveler is a document that moves through stations, tracking work progress.

```go
// internal/traveler/traveler.go
package traveler

import "time"

type TravelerStatus string

const (
    TravelerPending   TravelerStatus = "pending"
    TravelerActive    TravelerStatus = "active"
    TravelerComplete  TravelerStatus = "complete"
    TravelerFailed    TravelerStatus = "failed"
    TravelerDeferred  TravelerStatus = "deferred"
    TravelerRework    TravelerStatus = "rework"  // Sent back for rework
)

type Traveler struct {
    ID              string         `json:"id"`
    StationID       string         `json:"station_id"`
    BeadID          string         `json:"bead_id"`
    SOPID           string         `json:"sop_id,omitempty"`      // Attached molecule/SOP
    Priority        int            `json:"priority"`
    Status          TravelerStatus `json:"status"`
    Deferred        bool           `json:"deferred"`
    Restart         bool           `json:"restart"`
    ReworkCount     int            `json:"rework_count"`          // Times sent back for rework
    ReworkReason    string         `json:"rework_reason,omitempty"`
    AttachedAt      time.Time      `json:"attached_at"`
    StartedAt       *time.Time     `json:"started_at,omitempty"`
    CompletedAt     *time.Time     `json:"completed_at,omitempty"`
    Result          string         `json:"result,omitempty"`
    Error           string         `json:"error,omitempty"`
}

type TravelerManager struct {
    client *beads.Client
    events *events.EventBus
    store  *store.Store
}

func NewTravelerManager(client *beads.Client, events *events.EventBus, store *store.Store) *TravelerManager

// Attach assigns work to a station
func (t *TravelerManager) Attach(stationID, beadID string, opts ...AttachOption) error

// AttachSOP assigns a molecule/SOP to a station
func (t *TravelerManager) AttachSOP(stationID, sopID string, opts ...AttachOption) error

// Detach removes work from a station
func (t *TravelerManager) Detach(stationID string) error

// GetTraveler retrieves the current traveler for a station
func (t *TravelerManager) GetTraveler(stationID string) (*Traveler, error)

// Complete marks the traveler as complete
func (t *TravelerManager) Complete(stationID, result string) error

// Fail marks the traveler as failed
func (t *TravelerManager) Fail(stationID, reason string) error

// Rework sends the traveler back to a station with feedback
func (t *TravelerManager) Rework(stationID, reason string) error

type AttachOption func(*Traveler)
func WithPriority(p int) AttachOption
func WithDefer() AttachOption
func WithRestart() AttachOption
```

### 5. DAG Workflow Engine (Process Engineer)

The Process Engineer evaluates workflow dependencies and queues ready steps.

```go
// internal/workflow/dag.go
package workflow

import (
    "time"
    "sync"
)

// StepStatus tracks step execution state
type StepStatus string

const (
    StepPending    StepStatus = "pending"
    StepQueued     StepStatus = "queued"      // Ready to run (deps met)
    StepRunning    StepStatus = "running"
    StepWaiting    StepStatus = "waiting"     // Waiting for deps
    StepDone       StepStatus = "done"
    StepFailed     StepStatus = "failed"
    StepSkipped    StepStatus = "skipped"
)

// Step represents a single operation in a workflow
type Step struct {
    ID           string     `json:"id"`
    Name         string     `json:"name"`
    Description  string     `json:"description,omitempty"`
    Assignee     string     `json:"assignee,omitempty"`     // Preferred station type
    Dependencies []string   `json:"dependencies,omitempty"` // Step IDs this depends on
    Status       StepStatus `json:"status"`
    Acceptance   string     `json:"acceptance,omitempty"`   // Acceptance criteria
    Gate         string     `json:"gate,omitempty"`         // Must pass before proceeding
    Timeout      int        `json:"timeout,omitempty"`      // Seconds
    MaxRetries   int        `json:"max_retries,omitempty"`
    Retries      int        `json:"retries"`
    StartedAt    *time.Time `json:"started_at,omitempty"`
    CompletedAt  *time.Time `json:"completed_at,omitempty"`
    Result       string     `json:"result,omitempty"`
    Error        string     `json:"error,omitempty"`
}

// SOP (Standard Operating Procedure) - was "Molecule"
type SOPStatus string

const (
    SOPPending  SOPStatus = "pending"
    SOPRunning  SOPStatus = "running"
    SOPComplete SOPStatus = "complete"
    SOPFailed   SOPStatus = "failed"
    SOPPaused   SOPStatus = "paused"
)

type SOP struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Description string     `json:"description,omitempty"`
    Steps       []*Step    `json:"steps"`
    Status      SOPStatus  `json:"status"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    IsWisp      bool       `json:"is_wisp,omitempty"`
}

// DAGEngine evaluates dependencies and determines what can run
type DAGEngine struct {
    events *events.EventBus
    store  *store.Store
    mu     sync.RWMutex
}

func NewDAGEngine(events *events.EventBus, store *store.Store) *DAGEngine

// CreateSOP creates a new SOP from steps
func (e *DAGEngine) CreateSOP(name string, steps []*Step) (*SOP, error)

// Evaluate returns all steps that are ready to run (dependencies met)
func (e *DAGEngine) Evaluate(sopID string) ([]*Step, error)

// QueueReady finds all ready steps and emits StepQueued events
func (e *DAGEngine) QueueReady(sopID string) error

// StartStep marks a step as running
func (e *DAGEngine) StartStep(sopID, stepID, stationID string) error

// CompleteStep marks a step done and triggers dependency evaluation
func (e *DAGEngine) CompleteStep(sopID, stepID, result string) error

// FailStep marks a step failed
func (e *DAGEngine) FailStep(sopID, stepID, reason string) error

// RetryStep retries a failed step
func (e *DAGEngine) RetryStep(sopID, stepID string) error

// GetRunningSteps returns all currently running steps
func (e *DAGEngine) GetRunningSteps(sopID string) ([]*Step, error)

// GetPendingSteps returns all pending steps (deps not met)
func (e *DAGEngine) GetPendingSteps(sopID string) ([]*Step, error)

// IsComplete checks if all steps are done
func (e *DAGEngine) IsComplete(sopID string) (bool, error)
```

### 6. Formulas & Protomolecules

```go
// internal/workflow/formula.go
package workflow

// FormulaStep defines a step in a TOML formula
type FormulaStep struct {
    Name         string            `toml:"name"`
    Description  string            `toml:"description,omitempty"`
    Assignee     string            `toml:"assignee,omitempty"`
    Dependencies []string          `toml:"dependencies,omitempty"`
    Acceptance   string            `toml:"acceptance,omitempty"`
    Gate         string            `toml:"gate,omitempty"`
    Timeout      int               `toml:"timeout,omitempty"`
    MaxRetries   int               `toml:"max_retries,omitempty"`
    Variables    map[string]string `toml:"variables,omitempty"`
}

// Formula is a TOML recipe that cooks into a Protomolecule
type Formula struct {
    Name        string            `toml:"name"`
    Description string            `toml:"description,omitempty"`
    Variables   map[string]string `toml:"variables,omitempty"`
    Steps       []FormulaStep     `toml:"steps"`
}

// LoadFormula loads a formula from a TOML file
func LoadFormula(path string) (*Formula, error)

// ParseFormula parses a formula from string
func ParseFormula(data string) (*Formula, error)

// Cook converts a formula to a protomolecule
func (f *Formula) Cook() (*Protomolecule, error)

// CookWithVars converts with variable substitution
func (f *Formula) CookWithVars(vars map[string]string) (*Protomolecule, error)

// Protomolecule is a reusable SOP template
type Protomolecule struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Steps       []*Step   `json:"steps"`
    CreatedAt   time.Time `json:"created_at"`
}

// Instantiate creates a new SOP from this template
func (p *Protomolecule) Instantiate(vars map[string]string) (*SOP, error)

// InstantiateAsWisp creates an ephemeral SOP
func (p *Protomolecule) InstantiateAsWisp(vars map[string]string) (*SOP, error)
```

### 7. Production Planner (Scheduler)

The single source of truth for work dispatch. Eliminates race conditions.

```go
// internal/planner/planner.go
package planner

import (
    "context"
    "time"
)

type Planner struct {
    events         *events.EventBus
    store          *store.Store
    travelerMgr    *traveler.TravelerManager
    stationManager *station.Manager
    dagEngine      *workflow.DAGEngine
    client         *beads.Client
    director       *director.Director  // Reports to Director
}

func NewPlanner(
    events *events.EventBus,
    store *store.Store,
    travelerMgr *traveler.TravelerManager,
    stationManager *station.Manager,
    dagEngine *workflow.DAGEngine,
    client *beads.Client,
    director *director.Director,
) *Planner

// Start begins listening for events and dispatching work
func (p *Planner) Start(ctx context.Context) error

// Dispatch assigns a bead to a specific station
func (p *Planner) Dispatch(beadID, stationID string) error

// AutoDispatch finds an available station for the bead
func (p *Planner) AutoDispatch(beadID string) (string, error)

// DispatchBatch dispatches multiple beads to available stations
func (p *Planner) DispatchBatch(beadIDs []string) (map[string]string, error)

// Prioritize updates the priority of a queued bead
func (p *Planner) Prioritize(beadID string, priority int) error

// GetQueue returns the current work queue
func (p *Planner) GetQueue() ([]*QueueItem, error)

// RequeueForRework re-queues a job that failed quality check
func (p *Planner) RequeueForRework(beadID, reason string) error

// QueueItem represents a queued work item
type QueueItem struct {
    BeadID    string    `json:"bead_id"`
    Priority  int       `json:"priority"`
    QueuedAt  time.Time `json:"queued_at"`
    Status    string    `json:"status"`
    StationID string    `json:"station_id,omitempty"`
}
```

### 8. Station Manager (Sandbox Manager)

Manages isolated workspaces for operators.

```go
// internal/station/manager.go
package station

import (
    "context"
    "time"
)

type StationStatus string

const (
    StationIdle     StationStatus = "idle"
    StationBusy     StationStatus = "busy"
    StationOffline  StationStatus = "offline"
    StationError    StationStatus = "error"
)

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

type Manager struct {
    projectPath string
    events      *events.EventBus
    store       *store.Store
    tmux        *tmux.Manager
    stations    map[string]*Station
    maxStations int
}

func NewManager(projectPath string, events *events.EventBus, store *store.Store, tmux *tmux.Manager, maxStations int) *Manager

// Provision creates a new station with isolated worktree
func (m *Manager) Provision(name string) (*Station, error)

// Decommission removes a station and cleans up worktree
func (m *Manager) Decommission(stationID string) error

// Get retrieves a station by ID
func (m *Manager) Get(stationID string) (*Station, error)

// List returns all stations
func (m *Manager) List() []*Station

// GetAvailable returns stations that are idle
func (m *Manager) GetAvailable() []*Station

// SetBusy marks a station as busy
func (m *Manager) SetBusy(stationID, jobID string) error

// SetIdle marks a station as idle
func (m *Manager) SetIdle(stationID string) error

// CleanupWorktree removes the git worktree for a station
func (m *Manager) CleanupWorktree(stationID string) error

// CreateWorktree creates a git worktree for a station
func (m *Manager) CreateWorktree(stationID, branch string) error
```

### 9. Operator Pool

```go
// internal/operator/pool.go
package operator

import (
    "context"
    "time"
)

type OperatorStatus string

const (
    OperatorIdle     OperatorStatus = "idle"
    OperatorWorking  OperatorStatus = "working"
    OperatorDone     OperatorStatus = "done"
    OperatorFailed   OperatorStatus = "failed"
    OperatorStuck    OperatorStatus = "stuck"
    OperatorHandoff  OperatorStatus = "handoff"
)

type Operator struct {
    ID            string         `json:"id"`
    Name          string         `json:"name"`
    StationID     string         `json:"station_id"`
    Status        OperatorStatus `json:"status"`
    CurrentTask   string         `json:"current_task,omitempty"`
    ClaudeSession string         `json:"claude_session,omitempty"`
    StartedAt     time.Time      `json:"started_at"`
    LastHeartbeat time.Time      `json:"last_heartbeat"`
    CompletedAt   *time.Time     `json:"completed_at,omitempty"`
    Skills        []string       `json:"skills,omitempty"`
}

type Pool struct {
    stationManager *station.Manager
    events         *events.EventBus
    store          *store.Store
    tmux           *tmux.Manager
    client         *beads.Client
    operators      map[string]*Operator
}

func NewPool(
    stationManager *station.Manager,
    events *events.EventBus,
    store *store.Store,
    tmux *tmux.Manager,
    client *beads.Client,
) *Pool

// Spawn creates a new operator at a station
func (p *Pool) Spawn(stationID string) (*Operator, error)

// SpawnWithTask creates an operator and assigns a task
func (p *Pool) SpawnWithTask(stationID, beadID string) (*Operator, error)

// Get retrieves an operator by ID
func (p *Pool) Get(operatorID string) (*Operator, error)

// List returns all operators
func (p *Pool) List() []*Operator

// Decommission gracefully stops an operator
func (p *Pool) Decommission(operatorID string) error

// Handoff gracefully restarts an operator with context transfer
func (p *Pool) Handoff(operatorID string, workOnTraveler bool) error

// SendHeartbeat updates the operator's last heartbeat
func (p *Pool) SendHeartbeat(operatorID string) error

// GetStuck returns operators that haven't sent heartbeat recently
func (p *Pool) GetStuck(timeout time.Duration) []*Operator
```

### 10. Work Cells (Parallel Execution Groups)

```go
// internal/workcell/workcell.go
package workcell

import (
    "context"
    "time"
)

type WorkCellStatus string

const (
    WorkCellStaging  WorkCellStatus = "staging"
    WorkCellActive   WorkCellStatus = "active"
    WorkCellComplete WorkCellStatus = "complete"
    WorkCellFailed   WorkCellStatus = "failed"
)

type WorkCell struct {
    ID           string         `json:"id"`
    Name         string         `json:"name"`
    Status       WorkCellStatus `json:"status"`
    Stations     []string       `json:"stations"`
    TargetBeads  []string       `json:"target_beads"`
    CreatedAt    time.Time      `json:"created_at"`
    StartedAt    *time.Time     `json:"started_at,omitempty"`
    CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

type Manager struct {
    stationManager *station.Manager
    planner        *planner.Planner
    events         *events.EventBus
    store          *store.Store
}

func NewManager(
    stationManager *station.Manager,
    planner *planner.Planner,
    events *events.EventBus,
    store *store.Store,
) *Manager

// Create creates a new work cell with specified stations
func (m *Manager) Create(name string, stationIDs []string, beadIDs []string) (*WorkCell, error)

// Activate starts parallel execution on all stations
func (m *Manager) Activate(cellID string) error

// Status returns current status of a work cell
func (m *Manager) Status(cellID string) (*WorkCell, error)

// Disperse stops all stations in the cell
func (m *Manager) Disperse(cellID string) error

// WaitForComplete blocks until all work is done or context cancelled
func (m *Manager) WaitForComplete(ctx context.Context, cellID string) error
```

### 11. Final Assembly (Merge Station)

```go
// internal/assembly/assembly.go
package assembly

import (
    "context"
    "time"
)

type MergeStatus string

const (
    MergePending    MergeStatus = "pending"
    MergeChecking   MergeStatus = "checking"   // Checking for conflicts
    MergeReady      MergeStatus = "ready"      // No conflicts
    MergeConflicted MergeStatus = "conflicted"
    MergeMerging    MergeStatus = "merging"
    MergeComplete   MergeStatus = "complete"
    MergeFailed     MergeStatus = "failed"
)

type MergeRequest struct {
    ID           string       `json:"id"`
    BeadID       string       `json:"bead_id"`
    StationID    string       `json:"station_id"`
    Branch       string       `json:"branch"`
    Status       MergeStatus  `json:"status"`
    Priority     int          `json:"priority"`
    Conflicts    []string     `json:"conflicts,omitempty"`
    SubmittedAt  time.Time    `json:"submitted_at"`
    MergedAt     *time.Time   `json:"merged_at,omitempty"`
    Error        string       `json:"error,omitempty"`
}

type Assembly struct {
    projectPath string
    events      *events.EventBus
    store       *store.Store
    client      *beads.Client
}

func NewAssembly(projectPath string, events *events.EventBus, store *store.Store, client *beads.Client) *Assembly

// Submit adds a merge request to the queue
func (a *Assembly) Submit(beadID, stationID, branch string) error

// CheckConflicts pre-checks for merge conflicts
func (a *Assembly) CheckConflicts(mrID string) ([]string, error)

// CanMergeParallel checks if MRs can be merged concurrently (no file overlap)
func (a *Assembly) CanMergeParallel(mrIDs []string) (bool, error)

// ProcessQueue processes all ready merge requests in parallel
func (a *Assembly) ProcessQueue(ctx context.Context) error

// Merge performs the actual merge
func (a *Assembly) Merge(mrID string) error

// GetQueue returns the current merge queue
func (a *Assembly) GetQueue() ([]*MergeRequest, error)

// Escalate marks an MR for human attention
func (a *Assembly) Escalate(mrID, reason string) error

// Start begins listening for merge events
func (a *Assembly) Start(ctx context.Context) error
```

### 12. Quality Inspector

```go
// internal/inspector/inspector.go
package inspector

import (
    "context"
    "time"
)

type Inspector struct {
    stationManager *station.Manager
    operatorPool   *operator.Pool
    events         *events.EventBus
    store          *store.Store
    tmux           *tmux.Manager
}

func NewInspector(
    stationManager *station.Manager,
    operatorPool *operator.Pool,
    events *events.EventBus,
    store *store.Store,
    tmux *tmux.Manager,
) *Inspector

// CheckOperators checks all operators for stuck states
func (i *Inspector) CheckOperators() ([]string, error)

// VerifyWork verifies completed work meets acceptance criteria
func (i *Inspector) VerifyWork(beadID string) (bool, string, error)

// HelpStuck attempts to help a stuck operator
func (i *Inspector) HelpStuck(operatorID string) error

// FailQuality marks work as failed and triggers rework
func (i *Inspector) FailQuality(beadID, reason string) error

// Start begins periodic inspection
func (i *Inspector) Start(ctx context.Context, interval time.Duration) error
```

### 13. Floor Supervisor

```go
// internal/supervisor/supervisor.go
package supervisor

import (
    "context"
    "time"
)

type Supervisor struct {
    events      *events.EventBus
    store       *store.Store
    inspector   *inspector.Inspector
    tmux        *tmux.Manager
}

func NewSupervisor(
    events *events.EventBus,
    store *store.Store,
    inspector *inspector.Inspector,
    tmux *tmux.Manager,
) *Supervisor

// CoordinateHandoff coordinates graceful handoff between operators
func (s *Supervisor) CoordinateHandoff(fromStationID, toStationID string) error

// Start begins supervision
func (s *Supervisor) Start(ctx context.Context) error

// GetStatus returns current floor status
func (s *Supervisor) GetStatus() (*FloorStatus, error)

type FloorStatus struct {
    TotalStations    int       `json:"total_stations"`
    ActiveStations   int       `json:"active_stations"`
    IdleStations     int       `json:"idle_stations"`
    ActiveOperators  int       `json:"active_operators"`
    StuckOperators   int       `json:"stuck_operators"`
    PendingMerges    int       `json:"pending_merges"`
    LastActivity     time.Time `json:"last_activity"`
}
```

### 14. Support Service (Maintenance + Reliability + Expeditor)

Combined service for v1 to reduce complexity.

```go
// internal/support/service.go
package support

import (
    "context"
    "time"
)

type TaskType string

const (
    TaskCleanup      TaskType = "cleanup"
    TaskHealthCheck  TaskType = "health_check"
    TaskNudge        TaskType = "nudge"
    TaskRecovery     TaskType = "recovery"
)

type Task struct {
    ID          string     `json:"id"`
    Type        TaskType   `json:"type"`
    Description string     `json:"description"`
    Status      string     `json:"status"`
    StartedAt   time.Time  `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Service struct {
    events  *events.EventBus
    store   *store.Store
    tmux    *tmux.Manager
    tasks   map[string]*Task
}

func NewService(events *events.EventBus, store *store.Store, tmux *tmux.Manager) *Service

// Start begins the support service
func (s *Service) Start(ctx context.Context) error

// RunCleanup cleans up completed stations and old data
func (s *Service) RunCleanup(ctx context.Context) error

// RunHealthCheck checks system health
func (s *Service) RunHealthCheck(ctx context.Context) (*HealthReport, error)

// Nudge sends a nudge to an operator
func (s *Service) Nudge(operatorID, message string) error

// NudgeAll sends nudge to all operators
func (s *Service) NudgeAll(message string) error

// RecoverExpiredLeases recovers work from expired leases
func (s *Service) RecoverExpiredLeases(ctx context.Context) error

type HealthReport struct {
    DatabaseOK      bool     `json:"database_ok"`
    TmuxOK          bool     `json:"tmux_ok"`
    BeadsOK         bool     `json:"beads_ok"`
    DiskSpaceMB     int64    `json:"disk_space_mb"`
    ActiveStations  int      `json:"active_stations"`
    ExpiredLeases   int      `json:"expired_leases"`
    Errors          []string `json:"errors,omitempty"`
}
```

### 15. Production Log (SQLite Store)

```go
// internal/store/store.go
package store

import (
    "database/sql"
    "time"
)

type Store struct {
    db *sql.DB
}

func NewStore(dbPath string) (*Store, error)

// Migrate runs database migrations
func (s *Store) Migrate() error

// Station operations
func (s *Store) SaveStation(station *station.Station) error
func (s *Store) GetStation(id string) (*station.Station, error)
func (s *Store) ListStations() ([]*station.Station, error)

// Operator operations
func (s *Store) SaveOperator(op *operator.Operator) error
func (s *Store) GetOperator(id string) (*operator.Operator, error)
func (s *Store) ListOperators() ([]*operator.Operator, error)
func (s *Store) UpdateHeartbeat(operatorID string) error
func (s *Store) GetStuckOperators(timeout time.Duration) ([]*operator.Operator, error)

// Lease operations (for crash recovery)
func (s *Store) AcquireLease(resourceType, resourceID, ownerID string, ttl time.Duration) (*Lease, error)
func (s *Store) ReleaseLease(leaseID string) error
func (s *Store) GetExpiredLeases() ([]*Lease, error)
func (s *Store) RenewLease(leaseID string, ttl time.Duration) error

type Lease struct {
    ID           string    `json:"id"`
    ResourceType string    `json:"resource_type"`
    ResourceID   string    `json:"resource_id"`
    OwnerID      string    `json:"owner_id"`
    AcquiredAt   time.Time `json:"acquired_at"`
    ExpiresAt    time.Time `json:"expires_at"`
}

// SOP execution state
func (s *Store) SaveSOP(sop *workflow.SOP) error
func (s *Store) GetSOP(id string) (*workflow.SOP, error)
func (s *Store) UpdateStepStatus(sopID, stepID string, status workflow.StepStatus) error

// Traveler operations
func (s *Store) SaveTraveler(t *traveler.Traveler) error
func (s *Store) GetTraveler(stationID string) (*traveler.Traveler, error)

// Event log (for replay/debugging)
func (s *Store) LogEvent(event events.Event) error
func (s *Store) GetEvents(since time.Time, eventType events.EventType) ([]events.Event, error)

// Dead letter queue
func (s *Store) SaveDeadLetter(event events.Event) error
func (s *Store) GetDeadLetter() ([]events.Event, error)

// Close closes the database
func (s *Store) Close() error
```

### 16. tmux Manager

```go
// internal/tmux/tmux.go
package tmux

type Session struct {
    Name       string `json:"name"`
    Window     int    `json:"window"`
    Pane       int    `json:"pane"`
    WorkingDir string `json:"working_dir"`
    Command    string `json:"command,omitempty"`
    Pid        int    `json:"pid,omitempty"`
}

type Manager struct {
    sessions map[string]*Session
}

func NewManager() (*Manager, error)

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(name, workDir string) (*Session, error)

// SendKeys sends keystrokes to a session
func (m *Manager) SendKeys(session string, keys string) error

// SendKeysToPane sends keystrokes to a specific pane
func (m *Manager) SendKeysToPane(session string, window, pane int, keys string) error

// CaptureOutput captures the current pane output
func (m *Manager) CaptureOutput(session string) (string, error)

// KillSession kills a tmux session
func (m *Manager) KillSession(name string) error

// ListSessions lists all tmux sessions
func (m *Manager) ListSessions() []*Session

// HasSession checks if a session exists
func (m *Manager) HasSession(name string) bool

// RenameSession renames a session
func (m *Manager) RenameSession(oldName, newName string) error

// SplitPane splits a pane
func (m *Manager) SplitPane(session string, window, pane int, horizontal bool) (int, error)
```

### 17. Plant Director

```go
// internal/director/director.go
package director

import (
    "context"
    "time"
)

type FactoryStatus struct {
    Running        bool              `json:"running"`
    Stations       []StationStatus   `json:"stations"`
    ActiveJobs     int               `json:"active_jobs"`
    PendingBatches int               `json:"pending_batches"`
    LastActivity   time.Time         `json:"last_activity"`
    Uptime         time.Duration     `json:"uptime"`
}

type StationStatus struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Status    string `json:"status"`
    CurrentJob string `json:"current_job,omitempty"`
}

type Director struct {
    planner        *planner.Planner
    stationManager *station.Manager
    supervisor     *supervisor.Supervisor
    supportService *support.Service
    events         *events.EventBus
    store          *store.Store
    tmux           *tmux.Manager
    client         *beads.Client
    startedAt      time.Time
}

func NewDirector(
    planner *planner.Planner,
    stationManager *station.Manager,
    supervisor *supervisor.Supervisor,
    supportService *support.Service,
    events *events.EventBus,
    store *store.Store,
    tmux *tmux.Manager,
    client *beads.Client,
) *Director

// Start initializes and starts the factory
func (d *Director) Start(ctx context.Context) error

// Stop gracefully shuts down the factory
func (d *Director) Stop() error

// ReceiveTask receives a task from the user
func (d *Director) ReceiveTask(task string) (*batch.Batch, error)

// GetStatus returns current factory status
func (d *Director) GetStatus() (*FactoryStatus, error)

// RunBatch creates and runs a production batch
func (d *Director) RunBatch(name string, beadIDs []string) (*batch.Batch, error)

// CreateBatch creates a batch without starting it
func (d *Director) CreateBatch(name string, beadIDs []string) (*batch.Batch, error)

// Pause pauses the factory (only Director can do this)
func (d *Director) Pause() error

// Resume resumes the factory
func (d *Director) Resume() error

// Escalate escalates an issue to human attention
func (d *Director) Escalate(issue string) error
```

### 18. Batch Manager

```go
// internal/batch/batch.go
package batch

import "time"

type BatchStatus string

const (
    BatchStaging  BatchStatus = "staging"
    BatchRunning  BatchStatus = "running"
    BatchComplete BatchStatus = "complete"
    BatchFailed   BatchStatus = "failed"
    BatchPartial  BatchStatus = "partial"  // Some completed, some failed
)

type Batch struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Description string       `json:"description,omitempty"`
    Status      BatchStatus  `json:"status"`
    TrackedIDs  []string     `json:"tracked_ids"`
    WorkCells   []string     `json:"work_cells,omitempty"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    CompletedAt *time.Time   `json:"completed_at,omitempty"`
    Result      string       `json:"result,omitempty"`
}

type Manager struct {
    client  *beads.Client
    events  *events.EventBus
    store   *store.Store
}

func NewManager(client *beads.Client, events *events.EventBus, store *store.Store) *Manager

// Create creates a new batch
func (m *Manager) Create(name string, trackedIDs []string) (*Batch, error)

// Track returns the current state of a batch
func (m *Manager) Track(batchID string) (*Batch, error)

// Complete marks a batch as complete
func (m *Manager) Complete(batchID string, result string) error

// Fail marks a batch as failed
func (m *Manager) Fail(batchID string, reason string) error

// List returns all batches
func (m *Manager) List(filter BatchFilter) ([]*Batch, error)

// Dashboard returns batch summary for TUI
func (m *Manager) Dashboard() ([]*BatchSummary, error)

type BatchSummary struct {
    ID            string  `json:"id"`
    Name          string  `json:"name"`
    Status        string  `json:"status"`
    TotalJobs     int     `json:"total_jobs"`
    CompletedJobs int     `json:"completed_jobs"`
    FailedJobs    int     `json:"failed_jobs"`
    Progress      float64 `json:"progress"`
}

type BatchFilter struct {
    Status BatchStatus
}
```

---

## Mail System

```go
// internal/mail/mail.go
package mail

import "time"

type MessageType string

const (
    MsgTask     MessageType = "task"
    MsgNotify   MessageType = "notify"
    MsgEscalate MessageType = "escalate"
    MsgReply    MessageType = "reply"
    MsgSystem   MessageType = "system"
)

type Message struct {
    ID        string      `json:"id"`
    From      string      `json:"from"`
    To        string      `json:"to"`
    Subject   string      `json:"subject"`
    Body      string      `json:"body"`
    Type      MessageType `json:"type"`
    Priority  int         `json:"priority"`
    Timestamp time.Time   `json:"timestamp"`
    Read      bool        `json:"read"`
}

type Service struct {
    client *beads.Client
}

func NewService(client *beads.Client) *Service
func (m *Service) Send(msg *Message) error
func (m *Service) Receive(stationID string) ([]*Message, error)
func (m *Service) MarkRead(stationID, messageID string) error
func (m *Service) Broadcast(from string, subject, body string) error
```

---

## Factory Universal Propulsion Principle (FUPP)

Operators follow FUPP:

1. **Check Traveler**: Operator checks their traveler for attached work
2. **Execute Immediately**: If work found, execute without confirmation
3. **Report Completion**: Update bead status and emit completion event
4. **Wait for Instructions**: If no work, check mail and wait

### Support Service Nudge

When operators don't follow FUPP automatically:

1. **Startup Poke**: Operator gets nudged 30-60 seconds after starting
2. **Heartbeat**: Operators send periodic heartbeats
3. **Seance**: Current operator can communicate with predecessor via `/resume`

---

## Nondeterministic Idempotence (NDI)

FactoryAI operates on the principle of Nondeterministic Idempotence:

- Work is expressed as SOPs (workflows)
- Each step is executed by superintelligent AI
- Workflows are durable - survive crashes and restarts
- Operators can self-correct using acceptance criteria
- Eventual completion is guaranteed as long as operators keep trying

---

## CLI Commands

```bash
# Factory management
factory init                        # Initialize a new factory
factory status                      # Show factory status
factory boot                        # Start all stations
factory shutdown                    # Graceful shutdown
factory pause                       # Pause factory (Director only)
factory resume                      # Resume factory

# Stations
factory station add <name>          # Provision a new station
factory station list                # List all stations
factory station remove <id>         # Decommission a station
factory station status <id>         # Show station status

# Operators
factory operator spawn <station>    # Spawn an operator at a station
factory operator list               # List all operators
factory operator status <id>        # Show operator status
factory operator decommission <id>  # Decommission an operator

# Work Cells
factory cell create <name> <stations...>  # Create a work cell
factory cell activate <cell-id>           # Activate parallel execution
factory cell status <cell-id>             # Show cell status
factory cell disperse <cell-id>           # Disperse cell

# Work management (via beads CLI)
factory job create <title>          # Create a job ticket (bead)
factory job list                    # List job tickets
factory job show <id>               # Show ticket details
factory job close <id>              # Close a ticket
factory job epic <id>               # Convert to epic
factory job add-child <parent> <child>  # Add child to epic

factory traveler attach <station> <job>     # Attach work to station
factory traveler attach-sop <station>  # Attach SOP to station
factory traveler show <station>             # Show station's traveler
factory traveler clear <station>            # Clear station's traveler

factory batch create <name> <jobs...>  # Create batch
factory batch status <id>              # Show batch status
factory batch list                     # List batches
factory batch dashboard                # Show batch dashboard (TUI)

# SOPs & Formulas
factory formula load <path>         # Load a formula
factory formula list                # List available formulas
factory formula cook <name>         # Cook formula to protomolecule

factory sop create <name>           # Create a SOP
factory sop instantiate <proto-id>  # Instantiate protomolecule
factory sop status <id>             # Show SOP status
factory sop advance <id>            # Advance to next step

# Communication
factory mail send <to> <subject> <body>  # Send mail to station
factory mail read                         # Read your mail
factory mail broadcast <subject> <body>   # Broadcast to all

# Execution
factory run --formula <path> --task "<task>"  # Run a formula
factory dispatch <job> <station>              # Dispatch work to station

# Support Service
factory nudge <operator>            # Nudge operator to check traveler
factory seance <operator>           # Talk to operator's predecessor
factory health                      # Run health check
factory cleanup                     # Run cleanup

# Merge Queue
factory mq list                     # List merge queue
factory mq status                   # Show merge queue status
factory mq escalate <mr-id>         # Escalate MR

# Roles
factory role start <role>           # Start a role agent
factory role stop <role>            # Stop a role agent
factory role list                   # List all roles and their status
```

---

## File Structure

```
internal/
├── events/
│   └── bus.go              # Event bus (Andon Board)
├── beads/
│   ├── client.go           # Beads CLI client
│   └── types.go            # Bead type definitions
├── workflow/
│   ├── dag.go              # DAG workflow engine
│   ├── formula.go          # TOML formula definitions
│   └── protomolecule.go    # Template SOPs
├── traveler/
│   └── traveler.go         # Traveler (work order) management
├── mail/
│   └── mail.go             # Inter-agent messaging
├── batch/
│   └── batch.go            # Batch management + dashboard
├── operator/
│   └── pool.go             # Operator pool
├── workcell/
│   └── workcell.go         # Work cell management
├── station/
│   └── manager.go          # Station + worktree management
├── assembly/
│   └── assembly.go         # Merge Queue manager
├── inspector/
│   └── inspector.go        # Quality inspector
├── supervisor/
│   └── supervisor.go       # Floor supervisor
├── support/
│   └── service.go          # Combined support service (maintenance + reliability + expeditor)
├── planner/
│   └── planner.go          # Production planner (scheduler)
├── director/
│   └── director.go         # Plant director
├── store/
│   ├── store.go            # SQLite store
│   └── migrations/         # Database migrations
│       └── 001_init.sql
├── tmux/
│   └── tmux.go             # tmux session management
├── config/
│   └── config.go           # Config loading
├── factory/
│   └── factory.go          # Factory orchestrator
└── tui/
    ├── model.go            # TUI model
    ├── update.go           # TUI update
    ├── view.go             # TUI view
    └── dashboard.go        # Batch dashboard view

cmd/factory/
└── main.go                 # CLI entry point

formulas/                    # Workflow recipes
├── release.toml            # Release workflow
├── code-review.toml        # Code review workflow
├── feature.toml            # Feature implementation workflow
└── assembly/               # Assembly workflows
    ├── pre-check.toml
    └── merge.toml

configs/                     # Configuration files
├── factory.yaml            # Factory configuration
└── roles/                  # Role configurations
    ├── director.yaml
    ├── operator.yaml
    ├── inspector.yaml
    └── supervisor.yaml
```

---

## Database Schema

```sql
-- internal/store/migrations/001_init.sql

CREATE TABLE stations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    worktree_path TEXT,
    tmux_session TEXT,
    tmux_window INTEGER,
    tmux_pane INTEGER,
    current_job TEXT,
    operator_id TEXT,
    created_at DATETIME NOT NULL,
    last_activity DATETIME NOT NULL
);

CREATE TABLE operators (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    station_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_task TEXT,
    claude_session TEXT,
    started_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL,
    completed_at DATETIME,
    skills TEXT,  -- JSON array
    FOREIGN KEY (station_id) REFERENCES stations(id)
);

CREATE TABLE leases (
    id TEXT PRIMARY KEY,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    acquired_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE INDEX idx_leases_expires ON leases(expires_at);
CREATE INDEX idx_leases_resource ON leases(resource_type, resource_id);

CREATE TABLE sops (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    steps TEXT,  -- JSON array
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    completed_at DATETIME,
    is_wisp INTEGER DEFAULT 0
);

CREATE TABLE travelers (
    id TEXT PRIMARY KEY,
    station_id TEXT NOT NULL UNIQUE,
    bead_id TEXT NOT NULL,
    sop_id TEXT,
    priority INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    deferred INTEGER DEFAULT 0,
    restart INTEGER DEFAULT 0,
    rework_count INTEGER DEFAULT 0,
    rework_reason TEXT,
    attached_at DATETIME NOT NULL,
    started_at DATETIME,
    completed_at DATETIME,
    result TEXT,
    error TEXT,
    FOREIGN KEY (station_id) REFERENCES stations(id)
);

CREATE TABLE event_log (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    source TEXT,
    subject TEXT,
    payload TEXT  -- JSON
);

CREATE INDEX idx_events_timestamp ON event_log(timestamp);
CREATE INDEX idx_events_type ON event_log(type);

CREATE TABLE dead_letter (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    reason TEXT
);

CREATE TABLE batches (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    tracked_ids TEXT,  -- JSON array
    work_cells TEXT,   -- JSON array
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    completed_at DATETIME,
    result TEXT
);

CREATE TABLE merge_requests (
    id TEXT PRIMARY KEY,
    bead_id TEXT NOT NULL,
    station_id TEXT NOT NULL,
    branch TEXT NOT NULL,
    status TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    conflicts TEXT,  -- JSON array
    submitted_at DATETIME NOT NULL,
    merged_at DATETIME,
    error TEXT
);

CREATE INDEX idx_mr_status ON merge_requests(status);
```

---

## Dependencies

```go
// go.mod
module github.com/uttufy/FactoryAI

go 1.22

require (
    github.com/charmbracelet/bubbletea v0.26.0
    github.com/charmbracelet/lipgloss v0.11.0
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
    github.com/mattn/go-sqlite3 v1.14.22
    github.com/pelletier/go-toml/v2 v2.2.0
    github.com/spf13/cobra v1.8.1
    golang.org/x/sync v0.7.0
)
```

---

## Prerequisites

1. **beads CLI**: Install from github.com/steveyegge/beads
2. **tmux**: Required for session management
3. **claude CLI**: Claude Code binary from Anthropic
4. **Git**: Required for worktree management

---

## Verification

```bash
# Build
go build ./...

# Test
go test ./...

# Initialize factory
./factory init

# Verify beads integration
./factory job create "Test job"
./factory job list

# Provision a station
./factory station add station-1

# Create and dispatch work
./factory job create "Implement user authentication"
./factory dispatch job-123 station-1

# Create a batch
./factory batch create "Auth Feature" job-123 job-124 job-125

# Create a work cell
./factory cell create cell-1 station-1 station-2
./factory cell activate cell-1

# Check status
./factory status
./factory batch dashboard

# Run formula
./factory run --formula ./formulas/feature.toml --task "Build REST API"
```

---

## Future Enhancements

1. **Federation**: Remote stations on cloud providers
2. **GUI**: Web-based dashboard and control panel
3. **Plugin System**: Extensible plugins for custom workflows
4. **SOP Marketplace**: Marketplace for sharing formulas and SOPs
5. **Multi-Model Support**: Support for other AI models beyond Claude
6. **Typed Stations**: Specialized station types (coding, review, test)