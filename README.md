# FactoryAI

A manufacturing-plant-inspired multi-agent workspace manager in Go. FactoryAI orchestrates AI agents using factory floor metaphors with real manufacturing concepts: Stations, Operators, Travelers, SOPs, Quality Inspectors, Assembly Lines, and more.

Inspired by Steve Yegge's [Gas Town](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04).

## Overview

FactoryAI v1.0 is a complete rewrite implementing a production-grade multi-agent system with:

- **Beads CLI Integration** - Work item management via `beads` command
- **tmux Session Management** - Real-time station monitoring
- **SQLite Production Log** - Crash recovery and state persistence
- **DAG Workflow Engine** - SOP execution with dependency resolution
- **Event-Driven Architecture** - Andon Board pub/sub system
- **Git Worktree Isolation** - Parallel work in isolated environments
- **Merge Queue** - Conflict detection and parallel merging

## Features

### Core Architecture
- **Plant Director** - Single authority for orchestration
- **Production Planner** - Priority-based dispatch to available stations
- **Floor Supervisor** - Handoff coordination and monitoring
- **Quality Inspector** - Work verification and rework triggering
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
- **40+ CLI Commands** - Complete factory management
- **TUI Dashboard** - Real-time batch progress visualization
- **Mail System** - Inter-agent messaging

## Installation

```bash
# Clone the repository
git clone https://github.com/uttufy/FactoryAI.git
cd FactoryAI

# Build the binary
go build -o factory ./cmd/factory/main.go

# (Optional) Install to PATH
sudo mv factory /usr/local/bin/
```

## Requirements

- Go 1.22+
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
./factory batch create "batch-name" --beads bead1,bead2,bead3

# Check batch status
./factory batch status <batch-id>

# List batches
./factory batch list

# View dashboard
./factory batch dashboard <batch-id>
```

### Manage Stations

```bash
# Add a station
./factory station add --name "station-1" --git-ref main

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
./factory operator spawn --station <station-id> --role "developer"

# List operators
./factory operator list

# Show operator status
./factory operator status <operator-id>

# Decommission an operator
./factory operator decommission <operator-id>
```

### Work with Formulas

```bash
# Load a formula
./factory formula load --path ./formulas/feature.toml

# List formulas
./factory formula list

# Show formula status
./factory formula status <formula-id>
```

### Support Commands

```bash
# Run health check
./factory health

# Run cleanup
./factory cleanup

# Nudge an operator
./factory nudge --operator <operator-id> --message "Please continue"

# Nudge all operators
./factory nudge --all --message "Factory resuming"
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
| `internal/inspector` | Quality Inspector - work verification |
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
  name: "My Factory"
  description: "A production factory"

paths:
  project: "."
  formulas: "./formulas"
  roles: "./configs/roles"

stations:
  max_concurrent: 4
  default_branch: "main"

operators:
  heartbeat_interval: "30s"
  stuck_timeout: "5m"

events:
  buffer_size: 1000
  dead_letter_enabled: true
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
role = "Architect"
template = "Design a solution for: {task}"

[[steps]]
name = "implement"
role = "Developer"
depends_on = ["design"]
template = "Implement based on design: {context}"

[[steps]]
name = "review"
role = "Reviewer"
depends_on = ["implement"]
template = "Review the implementation: {context}"
```

### Code Review

```toml
name = "Code Review"
description = "Multi-perspective code review"

[[steps]]
name = "correctness"
role = "Senior Engineer"
template = "Review for correctness: {task}"

[[steps]]
name = "style"
role = "Style Reviewer"
template = "Review for style: {task}"

[[steps]]
name = "security"
role = "Security Expert"
template = "Review for security: {task}"
```

### Release Workflow

```toml
name = "Release"
description = "Complete release workflow"

[[steps]]
name = "test"
role = "QA Engineer"
template = "Run tests: {task}"

[[steps]]
name = "bump-version"
role = "Release Manager"
depends_on = ["test"]

[[steps]]
name = "tag"
role = "Release Manager"
depends_on = ["bump-version"]

[[steps]]
name = "push"
role = "DevOps Engineer"
depends_on = ["tag"]
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
- `job epic` - Create an epic
- `job add-child` - Add child to epic

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
- `formula load` - Load a formula
- `formula list` - List formulas
- `formula create` - Create a formula
- `formula status` - Show formula status

### Support Commands
- `health` - Run health check
- `cleanup` - Run cleanup
- `nudge` - Nudge an operator

### Merge Queue Commands
- `mq list` - List merge requests
- `mq status` - Show merge status
- `mq escalate` - Escalate merge issue

### Mail Commands
- `mail send` - Send mail
- `mail read` - Read mail
- `mail broadcast` - Broadcast mail

### Role Commands
- `role start` - Start a role
- `role stop` - Stop a role
- `role list` - List roles

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
- `StepStarted` - Processing step started
- `StepCompleted` - Processing step completed
- `StepFailed` - Processing step failed
- `StationReady` - Station ready for work
- `StationOffline` - Station went offline
- `OperatorStuck` - Operator detected as stuck
- `OperatorHandoff` - Operator handoff in progress
- `MergeReady` - Work ready for merge
- `MergeStarted` - Merge started
- `MergeCompleted` - Merge completed
- `MergeConflict` - Merge conflict detected
- `HealthOK` - Health check passed
- `CleanupDone` - Cleanup completed
- `QualityFailed` - Quality check failed
- `ReworkNeeded` - Rework required

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
