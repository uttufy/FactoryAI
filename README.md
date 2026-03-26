# FactoryAI

A manufacturing-plant-inspired multi-agent workspace manager in Go. FactoryAI orchestrates AI agents using factory floor metaphors with real manufacturing concepts: Stations, Operators, Travelers, SOPs, Quality Inspectors, Assembly Lines, and more.

Inspired by Steve Yegge's [Gas Town](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04).

## Overview

FactoryAI v0.2 is a production-grade multi-agent system with:

- **Beads CLI Integration** - Work item management via `beads` command
- **tmux Session Management** - Real-time station monitoring
- **SQLite Production Log** - Crash recovery and state persistence
- **DAG Workflow Engine** - SOP execution with dependency resolution
- **Event-Driven Architecture** - Andon Board pub/sub system
- **Git Worktree Isolation** - Parallel work in isolated environments
- **Merge Queue** - Conflict detection and parallel merging
- **Mail System** - Inter-agent messaging
- **Role Management** - Configurable agent roles

## Features

### Core Architecture
- **Plant Director** - Single authority for orchestration
- **Production Planner** - Priority-based dispatch to available stations
- **Floor Supervisor** - Handoff coordination and monitoring
- **Support Service** - Health checks, cleanup, and lease recovery
- **Final Assembly** - Merge queue with conflict detection

### Production Floor
- **Stations** - Git worktree-based isolated workspaces
- **Operators** - AI agents with heartbeat monitoring
- **Travelers** - Work order tracking through stations
- **Work Cells** - Parallel execution groups
- **Batches** - Production batch tracking

### Workflow System
- **Formulas** - TOML-based workflow recipes
- **SOPs** - Standard Operating Procedures with DAG dependencies
- **Protomolecules** - Template-based SOP generation
- **Variable Substitution** - Dynamic workflow customization

### User Interface
- **46+ CLI Commands** - Complete factory management
- **TUI Dashboard** - Real-time batch progress visualization
- **Mail System** - Inter-agent messaging

## Installation

```bash
# Clone the repository
git clone https://github.com/uttufy/FactoryAI.git
cd FactoryAI

# Build the binary
go build -o factory ./cmd/factory/...

# (Optional) Install to PATH
sudo mv factory /usr/local/bin/
```

## Requirements

- Go 1.24+
- [Claude CLI](https://claude.ai/code) installed and in PATH
- [Beads CLI](https://github.com/steveyegge/beads) installed and in PATH
- tmux (for session management)
- SQLite3 (included via Go bindings)

## Quick Start

### Initialize a Factory

```bash
# Initialize factory in current directory
./factory init

# Or specify a project path
./factory init --project-path /path/to/project
```

### Boot the Factory

```bash
# Start all stations
./factory boot

# Check factory status
./factory status
```

### Create and Run Jobs

```bash
# Create a job
./factory job create "Implement feature X"

# List jobs
./factory job list

# Show job details
./factory job show <job-id>

# Close a job
./factory job close <job-id>
```

### Work with Batches

```bash
# Create a batch
./factory batch create "batch-name" bead1 bead2 bead3

# Check batch status
./factory batch status <batch-id>

# List batches
./factory batch list

# View dashboard
./factory batch dashboard
```

### Manage Stations

```bash
# Add a station
./factory station add --name "station-1"

# List stations
./factory station list

# Show station status
./factory station status <station-id>

# Remove a station
./factory station remove <station-id>
```

### Manage Operators

```bash
# Spawn an operator
./factory operator spawn --station <station-id>

# List operators
./factory operator list

# Show operator status
./factory operator status <operator-id>

# Decommission an operator
./factory operator decommission <operator-id>
```

### Work with Formulas

```bash
# List available formulas
./factory formula list

# Show formula details
./factory formula show feature

# Create a new formula
./factory formula create my-workflow

# Validate a formula
./factory formula validate formulas/feature.toml
```

### Work with SOPs

```bash
# List SOPs
./factory sop list

# Show SOP details
./factory sop show <sop-id>

# Execute an SOP
./factory sop execute <sop-id>
```

### Execution Commands

```bash
# Run a job immediately
./factory run <job-id>

# Dispatch a job to a station
./factory dispatch <job-id> <station-id>

# Generate a plan from a goal
./factory plan "Build a REST API"
```

### Support Commands

```bash
# Run health check
./factory support status

# View support logs
./factory support logs

# Attach support to a station
./factory support attach <station-id>
```

### Merge Queue Commands

```bash
# Show merge queue status
./factory merge status

# List pending merges
./factory merge list

# Approve and execute a merge
./factory merge approve <mr-id>

# Block a merge request
./factory merge block <mr-id> "reason"
```

### Mail Commands

```bash
# Send a message
./factory mail send <to> <subject> <body>

# Broadcast to all stations
./factory mail broadcast <subject> <body>

# List messages
./factory mail list
```

### Role Commands

```bash
# List available roles
./factory role list

# Set current operator role
./factory role set <role>

# Clear current role
./factory role clear
```

## Architecture

### Authority Hierarchy

```
┌─────────────────────────────────────────────────────────┐
│ CONTROL ROOM - SINGLE POINT OF AUTHORITY                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ PLANT DIRECTOR — SINGLE AUTHORITY               │   │
│  │ ════════════════════════════════════════════    │   │
│  │ Owns: Schedule decisions, user interface        │   │
│  │ Can: Override planner, pause factory, escalate  │   │
│  └───────────────────────┬─────────────────────────┘   │
│                          │ Commands                    │
│          ┌───────────────┼───────────────┐             │
│          ▼               ▼               ▼             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐   │
│  │ PLANNER      │ │ ENGINEER     │ │ SUPERVISOR   │   │
│  │ ════════════ │ │ ════════════ │ │ ════════════ │   │
│  │ Executes:    │ │ Executes:    │ │ Executes:    │   │
│  │ - Dispatch   │ │ - DAG eval   │ │ - Handoff    │   │
│  │ - Queue mgmt │ │ - Routing    │ │ - Monitor    │   │
│  └──────────────┘ └──────────────┘ └──────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Data Flow

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
│  └───────────────────────┬─────────────────────────┘   │
│                          │ Source of truth              │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │ PRODUCTION LOG (SQLite) — RUNTIME CACHE         │   │
│  │ ════════════════════════════════════════════    │   │
│  │ Authority: NOTHING (derived state only)         │   │
│  │ Contains: Leases, heartbeats, transient state   │   │
│  │ Rule: If conflict, BEADS WINS                   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Key Components

| Package | Description |
|---------|-------------|
| `internal/director` | Plant Director - top-level orchestrator |
| `internal/planner` | Production Planner - single dispatcher |
| `internal/supervisor` | Floor Supervisor - handoff coordination |
| `internal/support` | Support Service - health & cleanup |
| `internal/assembly` | Final Assembly - merge queue |
| `internal/station` | Station Manager - worktree provisioning |
| `internal/operator` | Operator Pool - agent spawning |
| `internal/traveler` | Traveler Manager - work order tracking |
| `internal/workcell` | Work Cell Manager - parallel groups |
| `internal/batch` | Batch Manager - production tracking |
| `internal/workflow` | DAG Workflow Engine - SOP execution |
| `internal/beads` | Beads CLI Client - work items |
| `internal/tmux` | tmux Manager - session management |
| `internal/store` | SQLite Store - production log |
| `internal/events` | Event Bus (Andon Board) - pub/sub |
| `internal/mail` | Mail System - inter-agent messaging |
| `internal/tui` | TUI Dashboard - real-time visualization |

## Configuration

### Factory Configuration

Located at `configs/factory.yaml`:

```yaml
factory:
  name: "Main Factory"
  description: "Primary FactoryAI instance"

  # Station limits
  max_stations: 10
  max_operators: 20

  # Default timeouts
  default_timeout: 3600  # 1 hour
  heartbeat_interval: 30  # seconds

  # Worktree configuration
  worktree_dir: ".factory"
  cleanup_on_exit: true

# Event Bus configuration
event_bus:
  buffer_size: 1000
  log_events: true
  dead_letter_enabled: true

# Database configuration
database:
  path: "./factory.db"
  connection_pool: 5

# tmux configuration
tmux:
  session_prefix: "factory-"
  base_window_name: "factory"

# Claude configuration
claude:
  binary_path: ""  # Empty means use PATH or CLAUDE_BIN env var
  model: "claude-opus-4-6"  # Default model
```

### Role Configurations

Located at `configs/roles/`:

- `director.yaml` - Plant Director permissions
- `operator.yaml` - Operator capabilities
- `inspector.yaml` - Quality Inspector criteria
- `supervisor.yaml` - Supervisor authority

## Formulas

Formulas are TOML-based workflow recipes. Examples in `formulas/`:

### Feature Implementation

```toml
name = "Feature Implementation"
description = "Standard workflow for implementing features"

[[steps]]
name = "design"
description = "Design a solution for: {task}"
assignee = "architect"

[[steps]]
name = "implement"
description = "Implement based on design: {context}"
assignee = "developer"
dependencies = ["design"]

[[steps]]
name = "review"
description = "Review the implementation: {context}"
assignee = "reviewer"
dependencies = ["implement"]
```

### Code Review

```toml
name = "Code Review"
description = "Multi-perspective code review"

[[steps]]
name = "correctness"
description = "Review for correctness: {task}"
assignee = "Senior Engineer"

[[steps]]
name = "style"
description = "Review for style: {task}"
assignee = "Style Reviewer"

[[steps]]
name = "security"
description = "Review for security: {task}"
assignee = "Security Expert"
```

## CLI Commands

### Factory Management
- `init` - Initialize a factory
- `status` - Show factory status
- `boot` - Start all stations
- `shutdown` - Stop factory
- `pause` - Pause operations
- `resume` - Resume operations

### Station Commands
- `station add` - Add a station
- `station list` - List stations
- `station remove` - Remove a station
- `station status` - Show station status

### Operator Commands
- `operator spawn` - Spawn an operator
- `operator list` - List operators
- `operator status` - Show operator status
- `operator decommission` - Decommission an operator

### Work Cell Commands
- `cell create` - Create a work cell
- `cell activate` - Activate a work cell
- `cell status` - Show cell status
- `cell disperse` - Disperse a work cell

### Job Commands
- `job create` - Create a job
- `job list` - List jobs
- `job show` - Show job details
- `job close` - Close a job

### Traveler Commands
- `traveler attach` - Attach traveler to station
- `traveler show` - Show traveler status
- `traveler clear` - Clear traveler

### Batch Commands
- `batch create` - Create a batch
- `batch status` - Show batch status
- `batch list` - List batches
- `batch dashboard` - Show batch dashboard

### Formula Commands
- `formula create` - Create a formula
- `formula list` - List formulas
- `formula show` - Show formula details
- `formula validate` - Validate a formula

### SOP Commands
- `sop list` - List SOPs
- `sop show` - Show SOP details
- `sop execute` - Execute an SOP

### Execution Commands
- `run` - Run a job immediately
- `dispatch` - Dispatch a job to a station
- `plan` - Generate a plan from a goal

### Support Commands
- `support status` - Run health check
- `support logs` - View support logs
- `support attach` - Attach support to a station

### Merge Queue Commands
- `merge status` - Show merge queue status
- `merge list` - List pending merges
- `merge approve` - Approve and execute a merge
- `merge block` - Block a merge request

### Mail Commands
- `mail send` - Send a message
- `mail broadcast` - Broadcast to all stations
- `mail list` - List messages

### Role Commands
- `role list` - List available roles
- `role set` - Set current operator role
- `role clear` - Clear current role

## Environment Variables

```bash
# Override claude binary path
export CLAUDE_BIN=/path/to/claude

# Override factory config path
export FACTORY_CONFIG=/path/to/factory.yaml

# Override project path
export FACTORY_PROJECT=/path/to/project
```

## Event System (Andon Board)

FactoryAI uses an in-memory pub/sub event system for real-time communication:

### Event Types

- `JobCreated` - New job created
- `JobQueued` - Job queued for processing
- `JobStarted` - Processing started
- `JobCompleted` - Processing completed
- `JobFailed` - Processing failed
- `StepQueued` - Step queued for execution
- `StepStarted` - Step execution started
- `StepCompleted` - Step execution completed
- `StepFailed` - Step execution failed
- `StationReady` - Station ready for work
- `StationBusy` - Station is busy
- `StationOffline` - Station went offline
- `OperatorSpawned` - Operator spawned
- `OperatorIdle` - Operator is idle
- `OperatorStuck` - Operator detected as stuck
- `OperatorHandoff` - Operator handoff in progress
- `MergeReady` - Work ready for merge
- `MergeStarted` - Merge started
- `MergeCompleted` - Merge completed
- `MergeConflict` - Merge conflict detected
- `HealthOK` - Health check passed
- `CleanupDone` - Cleanup completed
- `FactoryShutdown` - Factory shutting down

### Subscriptions

Components subscribe to relevant events:

- **Director** - All events (monitoring)
- **Planner** - JobQueued, StepCompleted, StationReady
- **Supervisor** - OperatorStuck, StepFailed, OperatorHandoff
- **Support** - All events (read-only observer)
- **Assembly** - MergeReady, StepCompleted

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Steve Yegge's [Gas Town](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04) for the inspiration
- [Beads CLI](https://github.com/steveyegge/beads) for work item management
- [Claude Code](https://claude.ai/code) for the underlying AI agent
- [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI framework
