# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build the binary
go build -o factory ./cmd/factory/main.go

# Run the factory CLI
./factory [command]

# Install to PATH (optional)
sudo mv factory /usr/local/bin/
```

## Requirements

- Go 1.24+
- Claude CLI (`claude`) installed and in PATH
- Beads CLI (`beads`) for work item management
- tmux for session management

## Architecture Overview

FactoryAI is a multi-agent workspace manager that uses manufacturing factory metaphors to orchestrate parallel AI agents.

### Authority Hierarchy

1. **Plant Director** (`internal/director`) - Single authority for orchestration
   - Owns schedule decisions, user interface
   - Can override planner, pause factory, escalate issues

2. **Production Planner** (`internal/planner`) - Single dispatcher
   - Executes dispatch and queue management
   - Priority-based job scheduling

3. **Floor Supervisor** (`internal/supervisor`) - Handoff coordination
   - Monitors operator health and handoffs
   - Handles escalations

### Data Plane

1. **Beads CLI** (`internal/beads`) - MASTER SOURCE OF TRUTH
   - Job definitions, SOPs, final status
   - Git-backed, mutation only through beads CLI

2. **Production Log** (`internal/store`) - SQLite runtime cache
   - Leases, heartbeats, transient state
   - If conflict with Beads, Beads wins

### Core Components

| Package | Purpose |
|---------|---------|
| `internal/director` | Top-level orchestrator, single authority |
| `internal/planner` | Job dispatch and queue management |
| `internal/supervisor` | Operator handoff and monitoring |
| `internal/station` | Git worktree-based isolated workspaces |
| `internal/operator` | AI agent spawning and heartbeat monitoring |
| `internal/workflow` | DAG workflow engine, SOP execution |
| `internal/events` | Andon Board pub/sub event bus |
| `internal/store` | SQLite production log for crash recovery |
| `internal/tmux` | tmux session management |
| `internal/beads` | Beads CLI client for work items |

### Key Concepts

- **Stations**: Isolated git worktree workspaces with dedicated tmux sessions
- **Operators**: AI agents (Claude instances) with heartbeat monitoring
- **Travelers**: Work order tracking through stations
- **Formulas**: TOML-based workflow recipes (see `formulas/`)
- **SOPs**: Standard Operating Procedures with DAG dependencies
- **Protomolecules**: Reusable SOP templates

## Event System (Andon Board)

Event types are defined in `internal/events/bus.go`. Components subscribe to events via the EventBus:

- `EventJobCreated`, `EventJobQueued`, `EventJobStarted`, `EventJobCompleted`, `EventJobFailed`
- `EventStationReady`, `EventStationBusy`, `EventStationOffline`
- `EventStepQueued`, `EventStepStarted`, `EventStepCompleted`, `EventStepFailed`
- `EventMergeReady`, `EventMergeStarted`, `EventMergeCompleted`, `EventMergeConflict`
- `EventOperatorSpawned`, `EventOperatorIdle`, `EventOperatorStuck`, `EventOperatorHandoff`

## Configuration

Main config: `configs/factory.yaml`

Environment variables:
- `CLAUDE_BIN` - Override Claude binary path
- `FACTORY_CONFIG` - Override config path
- `FACTORY_PROJECT` - Override project path

## CLI Structure

The CLI uses Cobra with command groups defined in `cmd/factory/main.go`:
- Factory commands: `init`, `status`, `boot`, `shutdown`, `pause`, `resume`
- Station commands: `station add/list/remove/status`
- Operator commands: `operator spawn/list/status/decommission`
- Job commands: `job create/list/show/close/epic`
- Batch commands: `batch create/status/list/dashboard`
- Formula commands: `formula load/list/create/status`
- Support commands: `health`, `cleanup`, `nudge`
