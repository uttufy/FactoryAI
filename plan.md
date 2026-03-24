# FactoryAI - Low-Level Design (LLD)

## Overview

FactoryAI is a multi-agent workspace manager inspired by Gas Town, using a manufacturing factory metaphor. It orchestrates parallel AI agents working on software development tasks using the MEOW stack (Molecular Expression of Work).

**Key Integrations:**
- **Beads CLI**: Uses the actual `beads` CLI tool (github.com/steveyegge/beads) for work item management
- **tmux**: Primary UI and session management
- **Claude Code**: The underlying AI agent (`claude` binary)

## Factory Metaphor Mapping

| Gas Town Role | Factory Role | Description |
|---------------|--------------|-------------|
| Mayor | **Plant Manager** | Main orchestrator, receives tasks, dispatches work |
| Crew | **Floor Crew** | Long-lived agents per rig for interactive work with user |
| Polecats | **Workers** | Ephemeral workers spawned for specific tasks |
| Refinery | **Merge Station** | Manages Merge Queue, intelligently merges MRs |
| Witness | **Floor Monitor** | Monitors workers, helps unstick them |
| Deacon | **Shift Supervisor** | Daemon beacon, runs patrol loops, keeps factory running |
| Dogs | **Helper Crew** | Deacon's personal helpers for maintenance tasks |
| Boot | **Watchdog** | Special helper that checks on Deacon every 5 minutes |
| Town | **Factory** | The main workspace |
| Rigs | **Production Lines** | Projects/codebases being worked on |
| Hooks | **Work Orders** | Persistent task attachments on agents |
| Convoys | **Production Batches** | Tracked units of work delivery |
| Beads | **Job Tickets** | Individual work items |
| Wisps | **Ephemeral Tickets** | Temporary beads for orchestration (not persisted to Git) |
| Epics | **Work Packages** | Hierarchical beads with children |
| Molecules | **Assembly Instructions** | Workflows/sequences of tasks |
| Protomolecules | **Workflow Templates** | Reusable workflow blueprints |
| Formulas | **Workflow Recipes** | TOML definitions that cook into protomolecules |
| Swarms | **Worker Groups** | Groups of workers attacking work together |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Factory (Town)                                 │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                       Plant Manager (Mayor)                        │  │
│  │  - Receives user requests                                          │  │
│  │  - Dispatches work to Production Lines                             │  │
│  │  - Monitors overall factory status                                 │  │
│  │  - Manages Convoys                                                 │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐             │
│  │ Production     │  │ Production     │  │ Production     │             │
│  │ Line 1 (Rig)   │  │ Line 2 (Rig)   │  │ Line N (Rig)   │             │
│  │                │  │                │  │                │             │
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐ │             │
│  │ │Floor Crew  │ │  │ │Floor Crew  │ │  │ │Floor Crew  │ │             │
│  │ │(long-lived)│ │  │ │(long-lived)│ │  │ │(long-lived)│ │             │
│  │ └────────────┘ │  │ └────────────┘ │  │ └────────────┘ │             │
│  │                │  │                │  │                │             │
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐ │             │
│  │ │ Workers    │ │  │ │ Workers    │ │  │ │ Workers    │ │             │
│  │ │ (Polecats) │ │  │ │ (Polecats) │ │  │ │ (Polecats) │ │             │
│  │ └────────────┘ │  │ └────────────┘ │  │ └────────────┘ │             │
│  │                │  │                │  │                │             │
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐ │             │
│  │ │Floor Monitor│ │  │ │Floor Monitor│ │  │ │Floor Monitor│ │             │
│  │ │ (Witness)  │ │  │ │ (Witness)  │ │  │ │ (Witness)  │ │             │
│  │ └────────────┘ │  │ └────────────┘ │  │ └────────────┘ │             │
│  │                │  │                │  │                │             │
│  │ ┌────────────┐ │  │ ┌────────────┐ │  │ ┌────────────┐ │             │
│  │ │Merge Station│ │  │ │Merge Station│ │  │ │Merge Station│ │             │
│  │ │ (Refinery) │ │  │ │ (Refinery) │ │  │ │ (Refinery) │ │             │
│  │ └────────────┘ │  │ └────────────┘ │  │ └────────────┘ │             │
│  └────────────────┘  └────────────────┘  └────────────────┘             │
│                                                                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐             │
│  │ Shift          │  │ Helper Crew    │  │ Watchdog       │             │
│  │ Supervisor     │  │ (Dogs)         │  │ (Boot)         │             │
│  │ (Deacon)       │  │                │  │                │             │
│  └────────────────┘  └────────────────┘  └────────────────┘             │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │                        MEOW Stack                                    ││
│  │  Beads ─→ Epics ─→ Molecules ─→ Protomolecules ─→ Formulas          ││
│  │  (work)    (trees)  (workflows)  (templates)     (TOML recipes)     ││
│  └─────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Beads Integration

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

// Wisp operations
func (c *Client) CreateWisp(beadType, title string) (*Bead, error)
func (c *Client) Burn(id string) error
func (c *Client) Squash(id, summary string) error

// Epic operations
func (c *Client) CreateEpic(title string) (*Epic, error)
func (c *Client) AddChild(epicID, childID string) error
func (c *Client) GetChildren(epicID string) ([]*Bead, error)

// Molecule operations
func (c *Client) CreateMolecule(name string, steps []*Step) (*Molecule, error)
func (c *Client) InstantiateMolecule(protoID string, vars map[string]string) (*Molecule, error)

// Hook operations
func (c *Client) AttachHook(agentID, beadID string, opts ...AttachOption) error
func (c *Client) GetHook(agentID string) (*WorkOrder, error)
func (c *Client) ClearHook(agentID string) error

// Mail operations
func (c *Client) SendMail(from, to, subject, body string) error
func (c *Client) ReadMail(agentID string) ([]*Message, error)

// Cross-rig routing
func (c *Client) Route(prefix string) (*Client, error)
```

### 2. Bead Types

```go
// internal/beads/types.go
package beads

type BeadType string

const (
    BeadTask      BeadType = "task"       // Individual task
    BeadJobTicket BeadType = "job_ticket" // Work item
    BeadWorkOrder BeadType = "work_order" // Hook attachment
    BeadBatch     BeadType = "batch"      // Convoy/bundle
    BeadAssembly  BeadType = "assembly"   // Molecule/workflow
    BeadEvent     BeadType = "event"      // System event
    BeadRole      BeadType = "role"       // Role definition (pinned)
    BeadAgent     BeadType = "agent"      // Agent identity (pinned)
    BeadHook      BeadType = "hook"       // Agent's hook (pinned)
    BeadSwarm     BeadType = "swarm"      // Worker swarm (pinned)
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
    ProductionLine string                 `json:"production_line,omitempty"`
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
    Type           BeadType
    Status         BeadStatus
    Persistence    BeadPersistence
    Assignee       string
    ProductionLine string
    Labels         []string
    Pinned         *bool
}
```

### 3. Epics (Hierarchical Work)

```go
// internal/beads/epic.go
package beads

type Epic struct {
    *Bead
    Children  []string `json:"children"`
    Parallel  bool     `json:"parallel,omitempty"`
    Expand    bool     `json:"expand,omitempty"`
}

func (c *Client) NewEpic(title string) (*Epic, error)
func (c *Client) GetEpic(id string) (*Epic, error)
func (e *Epic) AddChild(client *Client, childID string) error
func (e *Epic) RemoveChild(client *Client, childID string) error
func (e *Epic) GetChildren(client *Client) ([]*Bead, error)
func (e *Epic) AllDone(client *Client) (bool, error)
```

### 4. Work Orders (Hooks)

```go
// internal/hooks/hooks.go
package hooks

import "time"

type WorkOrder struct {
    ID          string    `json:"id"`
    AgentID     string    `json:"agent_id"`
    BeadID      string    `json:"bead_id"`
    MoleculeID  string    `json:"molecule_id,omitempty"`
    Priority    int       `json:"priority"`
    Deferred    bool      `json:"deferred"`
    Restart     bool      `json:"restart"`
    AttachedAt  time.Time `json:"attached_at"`
    Status      string    `json:"status"`
}

type HookManager struct {
    client *beads.Client
}

func NewHookManager(client *beads.Client) *HookManager
func (h *HookManager) Attach(agentID, beadID string, opts ...AttachOption) error
func (h *HookManager) AttachMolecule(agentID, moleculeID string, opts ...AttachOption) error
func (h *HookManager) Detach(agentID string) error
func (h *HookManager) GetHookedWork(agentID string) (*WorkOrder, error)
func (h *HookManager) ClearHook(agentID string) error

type AttachOption func(*WorkOrder)
func WithPriority(p int) AttachOption
func WithDefer() AttachOption
func WithRestart() AttachOption
```

### 5. Mail System

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

type MailService struct {
    client *beads.Client
}

func NewMailService(client *beads.Client) *MailService
func (m *MailService) Send(msg *Message) error
func (m *MailService) Receive(agentID string) ([]*Message, error)
func (m *MailService) MarkRead(agentID, messageID string) error
func (m *MailService) Broadcast(from string, subject, body string) error
```

### 6. Convoys (Production Batches)

```go
// internal/convoys/convoys.go
package convoys

import "time"

type ConvoyStatus string

const (
    ConvoyStaging  ConvoyStatus = "staging"
    ConvoyRunning  ConvoyStatus = "running"
    ConvoyComplete ConvoyStatus = "complete"
    ConvoyFailed   ConvoyStatus = "failed"
    ConvoyPartial  ConvoyStatus = "partial"
)

type Convoy struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Description string       `json:"description,omitempty"`
    Status      ConvoyStatus `json:"status"`
    TrackedIDs  []string     `json:"tracked_ids"`
    Swarms      []string     `json:"swarms,omitempty"`
    Manager     string       `json:"manager,omitempty"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
    CompletedAt *time.Time   `json:"completed_at,omitempty"`
    Result      string       `json:"result,omitempty"`
}

type ConvoyManager struct {
    client *beads.Client
}

func NewConvoyManager(client *beads.Client) *ConvoyManager
func (c *ConvoyManager) Create(name string, trackedIDs []string) (*Convoy, error)
func (c *ConvoyManager) Track(convoyID string) (*Convoy, error)
func (c *ConvoyManager) Complete(convoyID string, result string) error
func (c *ConvoyManager) Fail(convoyID string, reason string) error
func (c *ConvoyManager) List(filter ConvoyFilter) ([]*Convoy, error)
func (c *ConvoyManager) Dashboard() ([]*Convoy, error)
```

### 7. MEOW Stack - Molecules

```go
// internal/molecules/molecule.go
package molecules

import "time"

type StepStatus string

const (
    StepPending  StepStatus = "pending"
    StepRunning  StepStatus = "running"
    StepWaiting  StepStatus = "waiting"
    StepDone     StepStatus = "done"
    StepFailed   StepStatus = "failed"
    StepSkipped  StepStatus = "skipped"
)

type Step struct {
    ID           string     `json:"id"`
    Name         string     `json:"name"`
    Description  string     `json:"description,omitempty"`
    Assignee     string     `json:"assignee,omitempty"`
    Dependencies []string   `json:"dependencies,omitempty"`
    Status       StepStatus `json:"status"`
    Acceptance   string     `json:"acceptance,omitempty"`
    Gate         string     `json:"gate,omitempty"`
    Timeout      int        `json:"timeout,omitempty"`
    StartedAt    *time.Time `json:"started_at,omitempty"`
    CompletedAt  *time.Time `json:"completed_at,omitempty"`
    Result       string     `json:"result,omitempty"`
    Error        string     `json:"error,omitempty"`
}

type MoleculeStatus string

const (
    MoleculePending  MoleculeStatus = "pending"
    MoleculeRunning  MoleculeStatus = "running"
    MoleculeComplete MoleculeStatus = "complete"
    MoleculeFailed   MoleculeStatus = "failed"
    MoleculePaused   MoleculeStatus = "paused"
)

type Molecule struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description,omitempty"`
    Steps       []*Step        `json:"steps"`
    CurrentStep int            `json:"current_step"`
    Status      MoleculeStatus `json:"status"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    CompletedAt *time.Time     `json:"completed_at,omitempty"`
    IsWisp      bool           `json:"is_wisp,omitempty"`
}

type MoleculeEngine struct {
    client *beads.Client
}

func NewMoleculeEngine(client *beads.Client) *MoleculeEngine
func (e *MoleculeEngine) Create(spec *Molecule) (*Molecule, error)
func (e *MoleculeEngine) Execute(moleculeID string) error
func (e *MoleculeEngine) Advance(moleculeID string) error
func (e *MoleculeEngine) GetCurrentStep(moleculeID string) (*Step, error)
func (e *MoleculeEngine) CompleteStep(moleculeID, stepID, result string) error
func (e *MoleculeEngine) FailStep(moleculeID, stepID, reason string) error
func (e *MoleculeEngine) Pause(moleculeID string) error
func (e *MoleculeEngine) Resume(moleculeID string) error
```

### 8. MEOW Stack - Formulas & Protomolecules

```go
// internal/molecules/formula.go
package molecules

type FormulaStep struct {
    Name         string            `toml:"name"`
    Description  string            `toml:"description,omitempty"`
    Assignee     string            `toml:"assignee,omitempty"`
    Dependencies []string          `toml:"dependencies,omitempty"`
    Acceptance   string            `toml:"acceptance,omitempty"`
    Gate         string            `toml:"gate,omitempty"`
    Timeout      int               `toml:"timeout,omitempty"`
    Variables    map[string]string `toml:"variables,omitempty"`
}

type Formula struct {
    Name        string            `toml:"name"`
    Description string            `toml:"description,omitempty"`
    Variables   map[string]string `toml:"variables,omitempty"`
    Steps       []FormulaStep     `toml:"steps"`
}

func LoadFormula(path string) (*Formula, error)
func ParseFormula(data string) (*Formula, error)
func (f *Formula) Cook() (*Protomolecule, error)
func (f *Formula) CookWithVars(vars map[string]string) (*Protomolecule, error)

// internal/molecules/protomolecule.go
package molecules

type Protomolecule struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Steps       []*Step   `json:"steps"`
    CreatedAt   time.Time `json:"created_at"`
}

func (p *Protomolecule) Instantiate(vars map[string]string) (*Molecule, error)
func (p *Protomolecule) InstantiateAsWisp(vars map[string]string) (*Molecule, error)
```

### 9. Workers (Polecats)

```go
// internal/workers/workers.go
package workers

import "time"

type WorkerStatus string

const (
    WorkerIdle     WorkerStatus = "idle"
    WorkerWorking  WorkerStatus = "working"
    WorkerDone     WorkerStatus = "done"
    WorkerFailed   WorkerStatus = "failed"
    WorkerHandoff  WorkerStatus = "handoff"
)

type Worker struct {
    ID            string       `json:"id"`
    Name          string       `json:"name"`
    ProductionLine string      `json:"production_line"`
    Status        WorkerStatus `json:"status"`
    CurrentTask   string       `json:"current_task,omitempty"`
    SessionID     string       `json:"session_id,omitempty"`
    ClaudeSession string       `json:"claude_session,omitempty"`
    StartedAt     time.Time    `json:"started_at"`
    CompletedAt   *time.Time   `json:"completed_at,omitempty"`
    CV            []string     `json:"cv,omitempty"`
}

type WorkerPool struct {
    productionLine string
    workers        map[string]*Worker
    maxWorkers     int
    tmux           *tmux.TmuxManager
    client         *beads.Client
}

func NewWorkerPool(productionLine string, maxWorkers int, tmux *tmux.TmuxManager, client *beads.Client) *WorkerPool
func (p *WorkerPool) Spawn(task string) (*Worker, error)
func (p *WorkerPool) SpawnWithMolecule(moleculeID string) (*Worker, error)
func (p *WorkerPool) GetWorker(id string) (*Worker, error)
func (p *WorkerPool) List() []*Worker
func (p *WorkerPool) Decommission(id string) error
func (p *WorkerPool) Handoff(id string, workOnHook bool) error
```

### 10. Swarms

```go
// internal/workers/swarm.go
package workers

import "time"

type SwarmStatus string

const (
    SwarmStaging  SwarmStatus = "staging"
    SwarmActive   SwarmStatus = "active"
    SwarmComplete SwarmStatus = "complete"
    SwarmFailed   SwarmStatus = "failed"
)

type Swarm struct {
    ID             string      `json:"id"`
    Name           string      `json:"name"`
    ProductionLine string      `json:"production_line"`
    Status         SwarmStatus `json:"status"`
    Workers        []string    `json:"workers"`
    TargetBeads    []string    `json:"target_beads"`
    CreatedAt      time.Time   `json:"created_at"`
    CompletedAt    *time.Time  `json:"completed_at,omitempty"`
}

type SwarmManager struct {
    pools  map[string]*WorkerPool
    client *beads.Client
}

func NewSwarmManager(client *beads.Client) *SwarmManager
func (s *SwarmManager) CreateSwarm(productionLine string, beadIDs []string, maxWorkers int) (*Swarm, error)
func (s *SwarmManager) Attack(swarmID string) error
func (s *SwarmManager) Status(swarmID string) (*Swarm, error)
func (s *SwarmManager) Disperse(swarmID string) error
```

### 11. Session Manager (tmux)

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

type TmuxManager struct {
    sessions map[string]*Session
}

func NewTmuxManager() (*TmuxManager, error)
func (t *TmuxManager) CreateSession(name, workDir string) (*Session, error)
func (t *TmuxManager) SendKeys(session string, keys string) error
func (t *TmuxManager) SendKeysToPane(session string, window, pane int, keys string) error
func (t *TmuxManager) CaptureOutput(session string) (string, error)
func (t *TmuxManager) KillSession(name string) error
func (t *TmuxManager) ListSessions() []*Session
func (t *TmuxManager) HasSession(name string) bool
func (t *TmuxManager) RenameSession(oldName, newName string) error
func (t *TmuxManager) SwapWindows(session string, srcWindow, dstWindow int) error
```

### 12. Merge Station (Refinery)

```go
// internal/refinery/refinery.go
package refinery

import "time"

type MergeRequest struct {
    ID          string     `json:"id"`
    BeadID      string     `json:"bead_id"`
    WorkerID    string     `json:"worker_id"`
    Branch      string     `json:"branch"`
    Status      string     `json:"status"`
    Priority    int        `json:"priority"`
    SubmittedAt time.Time  `json:"submitted_at"`
    MergedAt    *time.Time `json:"merged_at,omitempty"`
    Error       string     `json:"error,omitempty"`
}

type MergeQueue struct {
    ProductionLine string
    Queue          []*MergeRequest
    Current        *MergeRequest
}

type Refinery struct {
    productionLine string
    queue          *MergeQueue
    agent          agents.Agent
    tmux           *tmux.TmuxManager
    client         *beads.Client
}

func NewRefinery(productionLine string, agent agents.Agent, tmux *tmux.TmuxManager, client *beads.Client) *Refinery
func (r *Refinery) SubmitMR(beadID, workerID, branch string) error
func (r *Refinery) ProcessQueue() error
func (r *Refinery) GetCurrentMR() *MergeRequest
func (r *Refinery) Escalate(mrID, reason string) error
func (r *Refinery) RunPatrol() error
```

### 13. Floor Monitor (Witness)

```go
// internal/witness/witness.go
package witness

type Witness struct {
    productionLine string
    agent          agents.Agent
    tmux           *tmux.TmuxManager
    client         *beads.Client
    workerPool     *workers.WorkerPool
    refinery       *refinery.Refinery
}

func NewWitness(productionLine string, agent agents.Agent, tmux *tmux.TmuxManager, client *beads.Client, pool *workers.WorkerPool, ref *refinery.Refinery) *Witness
func (w *Witness) CheckWorkers() ([]string, error)
func (w *Witness) NudgeWorker(workerID string) error
func (w *Witness) CheckRefinery() error
func (w *Witness) RunPatrol() error
```

### 14. Shift Supervisor (Deacon)

```go
// internal/deacon/deacon.go
package deacon

import "time"

type Deacon struct {
    factory     *factory.Factory
    agent       agents.Agent
    tmux        *tmux.TmuxManager
    client      *beads.Client
    dogs        []*dogs.Dog
    patrol      *molecules.Molecule
    lastPatrol  time.Time
    patrolCount int
}

func NewDeacon(factory *factory.Factory, agent agents.Agent, tmux *tmux.TmuxManager, client *beads.Client) *Deacon
func (d *Deacon) RunPatrol() error
func (d *Deacon) DoYourJob() error
func (d *Deacon) Heartbeat() error
func (d *Deacon) SpawnDog(role dogs.DogRole) (*dogs.Dog, error)
func (d *Deacon) GetDogs() []*dogs.Dog
```

### 15. Helper Crew (Dogs)

```go
// internal/dogs/dogs.go
package dogs

import "time"

type DogRole string

const (
    DogBoot        DogRole = "boot"
    DogMaintenance DogRole = "maintenance"
    DogPlugin      DogRole = "plugin"
    DogInvestigate DogRole = "investigate"
)

type DogStatus string

const (
    DogSleeping DogStatus = "sleeping"
    DogWorking  DogStatus = "working"
    DogDone     DogStatus = "done"
)

type Dog struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Role        DogRole   `json:"role"`
    Status      DogStatus `json:"status"`
    CurrentTask string    `json:"current_task,omitempty"`
    SessionID   string    `json:"session_id,omitempty"`
    StartedAt   time.Time `json:"started_at"`
    WakeCount   int       `json:"wake_count"`
}

type DogPack struct {
    deaconID string
    dogs     map[string]*Dog
    agent    agents.Agent
    tmux     *tmux.TmuxManager
    client   *beads.Client
}

func NewDogPack(deaconID string, agent agents.Agent, tmux *tmux.TmuxManager, client *beads.Client) *DogPack
func (p *DogPack) SpawnDog(role DogRole) (*Dog, error)
func (p *DogPack) WakeDog(dogID string) error
func (p *DogPack) GetBoot() *Dog
func (p *DogPack) List() []*Dog
```

### 16. GUPP Nudge System

```go
// internal/nudge/nudge.go
package nudge

type NudgeService struct {
    tmux   *tmux.TmuxManager
    client *beads.Client
}

func NewNudgeService(tmux *tmux.TmuxManager, client *beads.Client) *NudgeService

// Nudge sends a tmux notification to wake up an agent
func (n *NudgeService) Nudge(agentID, message string) error

// NudgeAll sends nudge to all agents in a channel
func (n *NudgeService) NudgeAll(channel string, message string) error

// Seance allows talking to a previous session via /resume
func (n *NudgeService) Seance(currentAgentID string) (string, error)

// GetSessionID extracts Claude Code session ID from agent
func (n *NudgeService) GetSessionID(agentID string) (string, error)
```

### 17. Handoff Protocol

```go
// internal/handoff/handoff.go
package handoff

type HandoffManager struct {
    tmux   *tmux.TmuxManager
    hooks  *hooks.HookManager
    nudge  *nudge.NudgeService
    client *beads.Client
}

func NewHandoffManager(tmux *tmux.TmuxManager, hooks *hooks.HookManager, nudge *nudge.NudgeService, client *beads.Client) *HandoffManager

// Handoff gracefully restarts an agent
func (h *HandoffManager) Handoff(agentID string, opts ...HandoffOption) error

// HandoffWithWork hands off and attaches work to hook
func (h *HandoffManager) HandoffWithWork(agentID, beadID string) error

type HandoffOption func(*handoffConfig)
func WithRestart() HandoffOption
func WithMessage(msg string) HandoffOption
```

### 18. Plant Manager (Mayor)

```go
// internal/plantmanager/plantmanager.go
package plantmanager

import "context"

type FactoryStatus struct {
    Running         bool
    ProductionLines []ProductionLineStatus
    ActiveWorkers   int
    PendingConvoys  int
    LastActivity    time.Time
}

type ProductionLineStatus struct {
    Name          string
    ActiveWorkers int
    QueueLength   int
    Status        string
}

type PlantManager struct {
    factory       *factory.Factory
    mailService   *mail.MailService
    hookManager   *hooks.HookManager
    convoyManager *convoys.ConvoyManager
    tmuxManager   *tmux.TmuxManager
    nudgeService  *nudge.NudgeService
    handoffMgr    *handoff.HandoffManager
    client        *beads.Client
    deacon        *deacon.Deacon
}

func NewPlantManager(client *beads.Client, tmux *tmux.TmuxManager, agent agents.Agent) (*PlantManager, error)
func (p *PlantManager) Start(ctx context.Context) error
func (p *PlantManager) Stop() error
func (p *PlantManager) ReceiveTask(task string) (*convoys.Convoy, error)
func (p *PlantManager) DispatchWork(beadID string, target string) error
func (p *PlantManager) SlingWork(beadID string, target string, opts ...hooks.AttachOption) error
func (p *PlantManager) CheckStatus() (*FactoryStatus, error)
func (p *PlantManager) HandleInterrupt() error
func (p *PlantManager) RunConvoy(name string, trackedIDs []string) (*convoys.Convoy, error)
```

## Factory Universal Propulsion Principle (FUPP)

Like GUPP in Gas Town, FactoryAI follows FUPP:

1. **Check Hook**: Agent checks their hook for attached work
2. **Execute Immediately**: If work found, execute without confirmation
3. **Report Completion**: Update bead status and notify Plant Manager
4. **Wait for Instructions**: If no work, check mail and wait

### FUPP Nudge

Since agents don't always follow FUPP automatically:

1. **Startup Poke**: Agent gets nudged 30-60 seconds after starting
2. **Heartbeat**: Deacon sends periodic heartbeats to workers
3. **Seance**: Current agent can communicate with predecessor via `/resume`

## Patrol Loops

Patrol agents run workflows in loops with exponential backoff:

```go
type PatrolConfig struct {
    MinInterval    time.Duration
    MaxInterval    time.Duration
    BackoffFactor  float64
    CurrentBackoff time.Duration
}

func (p *PatrolAgent) RunPatrolLoop(ctx context.Context, config PatrolConfig) error
```

**Patrol Agents**:
- **Refinery**: Process merge queue
- **Witness**: Check on workers, help unstick them
- **Deacon**: Town-level orchestration, run plugins, coordinate handoffs

## Nondeterministic Idempotence (NDI)

FactoryAI operates on the principle of Nondeterministic Idempotence:

- Work is expressed as molecules (workflows)
- Each step is executed by superintelligent AI
- Workflows are durable - survive crashes and restarts
- Agents can self-correct using acceptance criteria
- Eventual completion is guaranteed as long as agents keep trying

## CLI Commands

```bash
# Factory management
factory init                        # Initialize a new factory
factory status                      # Show factory status
factory boot                        # Start all production lines
factory shutdown                    # Graceful shutdown

# Production Lines (Rigs)
factory line add <name> <path>      # Add a production line
factory line list                   # List production lines
factory line remove <name>          # Remove a production line
factory line status <name>          # Show line status

# Workers
factory worker spawn <line>         # Spawn a worker in a production line
factory worker list                 # List all workers
factory worker status <id>          # Show worker status
factory worker decommission <id>    # Decommission a worker

# Swarms
factory swarm create <line> <beads...>  # Create a swarm
factory swarm attack <swarm-id>         # Activate swarm
factory swarm status <swarm-id>         # Show swarm status
factory swarm disperse <swarm-id>       # Disperse swarm

# Work management (via beads CLI)
factory ticket create <title>       # Create a job ticket (bead)
factory ticket list                 # List job tickets
factory ticket show <id>            # Show ticket details
factory ticket close <id>           # Close a ticket
factory ticket epic <id>            # Convert to epic
factory ticket add-child <parent> <child>  # Add child to epic

factory hook attach <agent> <ticket>     # Attach work to agent
factory hook attach-mol <agent> <molecule> # Attach molecule to agent
factory hook show <agent>                # Show agent's hooked work
factory hook clear <agent>               # Clear agent's hook

factory convoy create <name> <tickets...>  # Create convoy
factory convoy status <id>                 # Show convoy status
factory convoy list                        # List convoys
factory convoy dashboard                   # Show convoy dashboard (TUI)

# Molecules & Formulas
factory formula load <path>         # Load a formula
factory formula list                # List available formulas
factory formula cook <name>         # Cook formula to protomolecule

factory molecule create <name>      # Create a molecule
factory molecule instantiate <proto-id>  # Instantiate protomolecule
factory molecule status <id>        # Show molecule status
factory molecule advance <id>       # Advance to next step

# Communication
factory mail send <to> <subject> <body>  # Send mail to agent
factory mail read                         # Read your mail
factory mail broadcast <subject> <body>   # Broadcast to all

# Execution
factory run --blueprint <path> --task "<task>"  # Run a factory blueprint
factory dispatch <ticket> <target>              # Dispatch work to target
factory sling <ticket> <target>                 # Sling work to agent's hook

# Handoff & Nudge
factory handoff <agent>            # Handoff agent (graceful restart)
factory nudge <agent>              # Nudge agent to check hook
factory seance <agent>             # Talk to agent's predecessor

# Merge Queue
factory mq list                    # List merge queue
factory mq status                  # Show merge queue status
factory mq escalate <mr-id>        # Escalate MR

# Roles
factory role start <role>          # Start a role agent
factory role stop <role>           # Stop a role agent
factory role list                  # List all roles and their status
```

## File Structure

```
internal/
├── beads/
│   ├── client.go           # Beads CLI client
│   ├── types.go            # Bead type definitions
│   └── epic.go             # Epic operations
├── molecules/
│   ├── molecule.go         # Molecule (workflow) types
│   ├── formula.go          # TOML formula definitions
│   └── protomolecule.go    # Template molecules
├── hooks/
│   └── hooks.go            # Work order/hook management
├── mail/
│   └── mail.go             # Inter-agent messaging
├── convoys/
│   └── convoys.go          # Convoy management + dashboard
├── workers/
│   ├── pool.go             # Worker pool
│   └── swarm.go            # Swarm management
├── refinery/
│   └── refinery.go         # Merge Queue manager
├── witness/
│   └── witness.go          # Worker monitoring
├── deacon/
│   ├── deacon.go           # Daemon beacon
│   └── patrol.go           # Patrol loop logic
├── dogs/
│   └── dogs.go             # Deacon's helpers (including Boot)
├── nudge/
│   └── nudge.go            # FUPP nudge + seance
├── handoff/
│   └── handoff.go          # Handoff protocol
├── tmux/
│   └── tmux.go             # tmux session management
├── plantmanager/
│   └── plantmanager.go     # Plant Manager (Mayor)
├── events/
│   └── events.go           # Event types
├── job/
│   └── job.go              # Job types
├── config/
│   └── config.go           # Config loading
├── agents/
│   ├── agent.go            # Agent interface
│   ├── claude.go           # Claude agent
│   └── registry.go         # Agent registry
├── factory/
│   └── factory.go          # Factory orchestrator
└── tui/
    ├── model.go            # TUI model
    ├── update.go           # TUI update
    ├── view.go             # TUI view
    └── dashboard.go        # Convoy dashboard view

cmd/factory/
└── main.go                 # CLI entry point

formulas/                    # Workflow recipes
├── release.toml            # Release workflow
├── code-review.toml        # Code review workflow
├── feature.toml            # Feature implementation workflow
└── patrol/                 # Patrol workflows
    ├── refinery.toml
    ├── witness.toml
    └── deacon.toml

blueprints/                  # Factory blueprints
├── coding_factory.yaml
├── research_factory.yaml
└── review_factory.yaml

configs/                     # Configuration files
├── factory.yaml            # Factory configuration
└── roles/                  # Role configurations
    ├── plant-manager.yaml
    ├── worker.yaml
    ├── refinery.yaml
    ├── witness.yaml
    ├── deacon.yaml
    └── dog.yaml
```

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
    github.com/pelletier/go-toml/v2 v2.2.0
    github.com/spf13/cobra v1.8.1
    golang.org/x/sync v0.7.0
)
```

## Prerequisites

1. **beads CLI**: Install from github.com/steveyegge/beads
2. **tmux**: Required for session management
3. **claude CLI**: Claude Code binary from Anthropic

## Verification

```bash
# Build
go build ./...

# Test
go test ./...

# Initialize factory
./factory init

# Verify beads integration
./factory ticket create "Test ticket"
./factory ticket list

# Add a production line
./factory line add myproject /path/to/project

# Create and dispatch work
./factory ticket create "Implement user authentication"
./factory sling ticket-123 worker-polecat-1

# Create a convoy
./factory convoy create "Auth Feature" ticket-123 ticket-124 ticket-125

# Start a swarm
./factory swarm create myproject ticket-123 ticket-124
./factory swarm attack swarm-1

# Check status
./factory status
./factory convoy dashboard

# Run blueprint
./factory run --blueprint ./blueprints/coding_factory.yaml --task "Build REST API"
```

## Future Enhancements

1. **Federation**: Remote workers on cloud providers
2. **GUI**: Web-based dashboard and control panel
3. **Plugin System**: Extensible plugins for custom workflows
4. **Mol Mall**: Marketplace for sharing formulas and molecules
5. **Multi-Model Support**: Support for other AI models beyond Claude