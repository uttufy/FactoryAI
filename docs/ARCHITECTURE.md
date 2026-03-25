# FactoryAI Architecture

## System Overview

FactoryAI is a multi-agent workspace manager that orchestrates AI agents working on software development tasks. It uses real manufacturing factory concepts and terminology, adapted for software production.

## Core Concepts

### Manufacturing Metaphors

| Factory Concept | Software Analog | Description |
|-----------------|-----------------|-------------|
| **Station** | Workspace | Isolated git worktree for work |
| **Operator** | AI Agent | Claude agent working at a station |
| **Traveler** | Work Order | Tracks work through stations |
| **SOP** | Workflow | Standard Operating Procedure (DAG) |
| **Formula** | Recipe | TOML-based workflow template |
| **Bead** | Work Item | Unit of work managed by beads CLI |
| **Batch** | Production Batch | Group of related work items |
| **Work Cell** | Team | Parallel execution group |
| **Assembly Line** | Pipeline | Sequential station processing |
| **Andon Board** | Event Bus | Pub/sub communication system |
| **Quality Inspector** | Validator | Verifies work quality |
| **Floor Supervisor** | Coordinator | Handoffs and monitoring |
| **Plant Director** | Orchestrator | Top-level authority |

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CONTROL ROOM                               │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     PLANT DIRECTOR                            │  │
│  │  - User interface (CLI)                                      │  │
│  │  - Task reception and batching                               │  │
│  │  - Factory lifecycle management                              │  │
│  │  - Escalation and override authority                         │  │
│  └────────────┬─────────────────────────────────┬───────────────┘  │
│               │                                 │                   │
│      ┌────────▼────────┐              ┌────────▼─────────┐         │
│      │ PRODUCTION      │              │ QUALITY          │         │
│      │ PLANNER         │              │ ENGINEER         │         │
│      │ ─────────────   │              │ ─────────────    │         │
│      │ - Job queue     │              │ - SOP execution  │         │
│      │ - Dispatch      │              │ - DAG eval       │         │
│      │ - Priority      │              │ - Routing        │         │
│      └────────┬────────┘              └────────┬─────────┘         │
│               │                                 │                   │
└───────────────┼─────────────────────────────────┼───────────────────┘
                │                                 │
        ┌───────▼─────────────────────────────────▼───────┐
        │              ANDON BOARD (Event Bus)            │
        │  ────────────────────────────────────────────   │
        │  - Pub/sub communication                        │
        │  - Event logging to SQLite                      │
        │  - Dead letter queue                            │
        └───┬──────────┬──────────┬──────────┬──────────┬──┘
            │          │          │          │          │
    ┌───────▼──┐ ┌────▼───┐ ┌───▼────┐ ┌───▼────┐ ┌───▼────┐
    │ FLOOR    │ │FINAL   │ │SUPPORT │ │MAIL    │ │TUI     │
    │SUPERVISOR│ │ASSEMBLY│ │SERVICE │ │SYSTEM  │ │DASHBOARD│
    │ ────────│ │───────│ │───────│ │───────│ │───────│
    │-Handoff  │ │-Merge  │ │-Health │ │-Inter- │ │-Real-  │
    │ Monitor  │ │-Conflict│ │-Cleanup│ │agent   │ │time    │
    │          │ │Detect  │ │-Recovery│ │msg     │ │progress│
    └───┬──────┘ └───┬────┘ └───┬────┘ └────────┘ └────────┘
        │            │           │
        └────────────┴───────────┴───────────┐
                                             │
        ┌────────────────────────────────────▼─────────────────────────┐
        │                    PRODUCTION FLOOR                           │
        │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
        │  │  STATION 1  │  │  STATION 2  │  │  STATION N  │          │
        │  │ ─────────── │  │ ─────────── │  │ ─────────── │          │
        │  │ - Worktree  │  │ - Worktree  │  │ - Worktree  │          │
        │  │ - tmux pane │  │ - tmux pane │  │ - tmux pane │          │
        │  │ - Operator  │  │ - Operator  │  │ - Operator  │          │
        │  │ - Traveler  │  │ - Traveler  │  │ - Traveler  │          │
        │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
        │         │                │                │                   │
        │         └────────────────┴────────────────┘                   │
        │                           │                                   │
        └───────────────────────────┼───────────────────────────────────┘
                                    │
        ┌───────────────────────────▼───────────────────────────────────┐
        │                    DATA PLANE                                 │
        │  ┌─────────────────┐    ┌─────────────────┐                   │
        │  │  BEADS (Git)    │    │  SQLite Log     │                   │
        │  │  ─────────────  │    │  ─────────────  │                   │
        │  │ - Master truth  │◄───│ - Runtime cache │                   │
        │  │ - Work items    │    │ - Leases        │                   │
        │  │ - Status        │    │ - Events        │                   │
        │  │ - Recovery      │    │ - Heartbeats    │                   │
        │  └─────────────────┘    └─────────────────┘                   │
        └───────────────────────────────────────────────────────────────┘
```

## Component Details

### Plant Director (`internal/director/`)

The single authority for factory orchestration.

**Responsibilities:**
- Receive and validate user tasks
- Create batches from tasks
- Manage factory lifecycle (boot, shutdown, pause, resume)
- Handle escalations
- Provide user interface

**Key Methods:**
- `ReceiveTask()` - Accept new task and create batch
- `Boot()` - Initialize all stations
- `Shutdown()` - Graceful shutdown
- `GetStatus()` - Query factory status

**Cannot:**
- Execute work directly (delegates to Planner/Engineer)
- Access stations directly (goes through Supervisor)

### Production Planner (`internal/planner/`)

Single dispatcher for work assignment.

**Responsibilities:**
- Maintain work queue with priority
- Dispatch beads to available stations
- Auto-dispatch based on station availability
- Handle rework queuing

**Key Methods:**
- `Enqueue()` - Add bead to queue
- `Dispatch()` - Assign bead to specific station
- `AutoDispatch()` - Find available station and dispatch
- `GetQueue()` - Query current queue

**Authority:**
- CAN: Dispatch work, manage queue
- CANNOT: Override decisions, execute work directly

### Quality Engineer / Workflow Engine (`internal/workflow/`)

Evaluates SOPs and routes work.

**Responsibilities:**
- Parse and evaluate SOP DAGs
- Execute steps in dependency order
- Handle step failures and retries
- Generate SOPs from Formulas

**Key Methods:**
- `Evaluate()` - Check if step can execute
- `Execute()` - Execute a step
- `CookFormula()` - Generate SOP from formula
- `GetStatus()` - Query step status

**Authority:**
- CAN: Execute steps, evaluate dependencies
- CANNOT: Dispatch work, override planner

### Floor Supervisor (`internal/supervisor/`)

Coordinates handoffs and monitors the floor.

**Responsibilities:**
- Coordinate operator handoffs between stations
- Monitor station health
- Detect stuck operators
- Trigger recovery actions

**Key Methods:**
- `CoordinateHandoff()` - Hand off operator between stations
- `MonitorOperators()` - Check for stuck operators
- `VerifyStations()` - Health check all stations
- `GetStatus()` - Query floor status

### Quality Inspector (`internal/inspector/`)

Verifies work quality.

**Responsibilities:**
- Validate completed work
- Detect stuck operators
- Trigger rework when quality fails

**Key Methods:**
- `Inspect()` - Validate work output
- `GetStuck()` - Detect stuck operators
- `RequestRework()` - Send work back for rework

### Support Service (`internal/support/`)

Combined maintenance, reliability, and expeditor.

**Responsibilities:**
- Health checks
- Cleanup of completed work
- Lease recovery for crash recovery
- Nudging stuck operators

**Key Methods:**
- `RunHealthCheck()` - Check system health
- `RunCleanup()` - Clean up old data
- `RecoverExpiredLeases()` - Recover from crashes
- `Nudge()` - Send message to operator

### Final Assembly (`internal/assembly/`)

Merge queue manager.

**Responsibilities:**
- Queue merge requests
- Detect merge conflicts
- Execute merges (sequential and parallel)
- Escalate unresolvable conflicts

**Key Methods:**
- `Submit()` - Add merge request to queue
- `CheckConflicts()` - Pre-check for conflicts
- `ProcessQueue()` - Process ready merges
- `Merge()` - Execute git merge

### Station Manager (`internal/station/`)

Manages workstation provisioning.

**Responsibilities:**
- Create git worktrees for isolation
- Manage tmux sessions for stations
- Track station status
- Decommission stations

**Key Methods:**
- `Create()` - Provision new station
- `Get()` - Get station details
- `SetIdle()` / `SetBusy()` - Update status
- `Decommission()` - Clean up station

### Operator Pool (`internal/operator/`)

Manages AI agents.

**Responsibilities:**
- Spawn operators at stations
- Monitor operator heartbeats
- Detect stuck operators
- Handle operator handoffs

**Key Methods:**
- `Spawn()` - Create new operator
- `GetOperatorByStation()` - Find operator at station
- `GetStuck()` - Find stuck operators
- `Handoff()` - Transfer operator between stations

### Traveler Manager (`internal/traveler/`)

Tracks work orders through stations.

**Responsibilities:**
- Attach travelers to stations
- Track progress through stations
- Handle completion and failure
- Manage rework loops

**Key Methods:**
- `Attach()` - Attach traveler to station
- `Complete()` - Mark traveler complete
- `Fail()` - Mark traveler failed
- `Rework()` - Requeue for rework

### Work Cell Manager (`internal/workcell/`)

Manages parallel execution groups.

**Responsibilities:**
- Create work cells
- Activate cells for parallel execution
- Disperse cells after completion

**Key Methods:**
- `Create()` - Create new work cell
- `Activate()` - Start parallel execution
- `Disperse()` - Clean up after completion

### Batch Manager (`internal/batch/`)

Manages production batches.

**Responsibilities:**
- Create batches from bead groups
- Track batch progress
- Provide dashboard data

**Key Methods:**
- `Create()` - Create new batch
- `GetStatus()` - Query batch status
- `GetDashboardData()` - Get dashboard info

### Beads Client (`internal/beads/`)

Wrapper around beads CLI.

**Responsibilities:**
- Create, get, update, delete beads
- Manage travelers
- Handle mail
- Execute molecules

**Key Methods:**
- `Create()` - Create new bead
- `Get()` - Get bead details
- `Update()` - Update bead
- `List()` - List beads

### tmux Manager (`internal/tmux/`)

Manages tmux sessions.

**Responsibilities:**
- Create tmux sessions
- Send keys to panes
- Capture output
- Kill sessions

**Key Methods:**
- `CreateSession()` - Create new session
- `SendKeys()` - Send input to pane
- `CaptureOutput()` - Get pane output
- `KillSession()` - Terminate session

### SQLite Store (`internal/store/`)

Production log database.

**Responsibilities:**
- Store runtime state
- Log events
- Manage leases
- Enable crash recovery

**Key Methods:**
- `CreateLease()` - Create lease for crash recovery
- `GetExpiredLeases()` - Find expired leases
- `LogEvent()` - Log event to database
- `GetEvents()` - Retrieve event log

### Event Bus (Andon Board) (`internal/events/`)

Pub/sub communication system.

**Responsibilities:**
- Event broadcasting
- Subscription management
- Dead letter queue
- Event logging

**Key Methods:**
- `Emit()` - Publish event
- `Subscribe()` - Subscribe to event type
- `SubscribeAll()` - Subscribe to all events

### Mail System (`internal/mail/`)

Inter-agent messaging.

**Responsibilities:**
- Send messages between operators
- Broadcast messages to all
- Retrieve messages

**Key Methods:**
- `Send()` - Send message
- `Receive()` - Get messages for operator
- `Broadcast()` - Send to all operators

### TUI Dashboard (`internal/tui/`)

Real-time visualization.

**Responsibilities:**
- Display batch progress
- Show station status
- Update in real-time via events

**Components:**
- `model.go` - TUI state
- `update.go` - Event handling
- `view.go` - Rendering

## Data Flow

### 1. Task Reception

```
User → Director → Create Bead → Create Batch
                     ↓
                  Planner.Enqueue()
```

### 2. Work Dispatch

```
Planner.ProcessQueue() → Find Available Station
                              ↓
                    Traveler.Attach()
                              ↓
                    Operator.Spawn()
                              ↓
                    Emit(StepStarted)
```

### 3. Work Execution

```
Operator → Claude Agent → Process Step
                            ↓
                      Inspector.Inspect()
                            ↓
                ┌───────────┴───────────┐
                ↓                       ↓
            PASS                     FAIL
                ↓                       ↓
        StepCompleted             Retry/Requeue
                ↓
        Traveler.Complete()
                ↓
        Merge.Submit()
```

### 4. Merge Phase

```
Merge.Submit() → CheckConflicts()
                       ↓
              ┌─────────┴─────────┐
              ↓                   ↓
          Conflicts           No Conflicts
              ↓                   ↓
         Escalate           ProcessQueue()
                                    ↓
                              Merge()
                                    ↓
                            MergeCompleted
```

## Authority Boundaries

### What Director CAN Do:
- Override planner decisions
- Pause/resume factory
- Escalate issues to humans
- Make policy decisions

### What Director CANNOT Do:
- Execute work directly
- Access stations directly
- Modify beads directly (uses beads client)

### What Planner CAN Do:
- Dispatch work to stations
- Manage priority queue
- Handle rework

### What Planner CANNOT Do:
- Override director
- Execute work directly
- Access operators directly

### What Supervisor CAN Do:
- Coordinate handoffs
- Monitor floor health
- Trigger recovery

### What Supervisor CANNOT Do:
- Dispatch work
- Execute work directly
- Override director/planner

## Event Flow

### Step Completion Flow

```
Operator finishes step
        ↓
Emit(StepCompleted)
        ↓
┌───────┴────────┐
↓                ↓
Planner          Inspector
│                │
│           Inspect()
│                │
│          ┌─────┴─────┐
│          ↓           ↓
│         PASS        FAIL
│          ↓           ↓
│     StepDone    RequestRework
│          ↓           ↓
└────────→←───────────┘
          ↓
    Traveler.Complete()
          ↓
    Next Step or Merge
```

## Crash Recovery

### Lease System

Each operator holds a lease with heartbeat:

```
Operator spawned → Create Lease (5 minute expiry)
                           ↓
                   Heartbeat every 30s
                           ↓
              ┌────────────┴────────────┐
              ↓                         ↓
        Heartbeat OK           Heartbeat Missed
              ↓                         ↓
        Renew Lease              Lease Expires
                                        ↓
                                  Support detects
                                        ↓
                                  Recover operator
```

### Recovery Process

```
Support detects expired lease
        ↓
Get operator's last state from beads
        ↓
Check bead status in beads
        ↓
┌───────┴────────┐
↓                ↓
Bead done      Bead in-progress
↓                   ↓
Ignore         Re-queue work
                   ↓
            Operator available
                   ↓
            Resume work
```

## Performance Considerations

### Concurrency

- Stations run in parallel (one per worktree)
- Steps within a station run sequentially
- Work cells enable parallel step groups
- Merges can run in parallel if no file overlap

### Scalability

- Max concurrent stations: Configurable
- Operator limit: One per station
- Work cells: Can group stations
- Batches: Can contain unlimited beads

### Bottlenecks

- Git operations during merge
- Claude API rate limits
- tmux session creation overhead
- SQLite write concurrency

## Security Considerations

### Isolation

- Each station: Separate git worktree
- Each operator: Separate tmux pane
- Beads: Git-backed with permissions

### Access Control

- Role-based config in `configs/roles/`
- Director has override authority
- Operators cannot modify system state

### Secrets

- Environment variables for sensitive data
- .env file support (gitignored)
- No secrets in beads (Git-backed)
