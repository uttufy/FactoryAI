# System Patterns: FactoryAI

## Architecture Overview

### Four-Layer Architecture
```
Layer 1: Interface & Observability (CLI, TUI, HTTP)
    ↓
Layer 2: Control Room (Event Bus, Director, Planner, Engineer)
    ↓
Layer 3: Production Floor (Stations, Operators, Supervisors, Assembly)
    ↓
Layer 4: Data Plane (Beads/Git, SQLite)
```

### Source of Truth Hierarchy
1. **Beads (Git-backed)** - MASTER SOURCE OF TRUTH
   - Job definitions, SOPs, final status
   - Mutation only through beads CLI
   - Git is the restore point for recovery

2. **SQLite (Production Log)** - RUNTIME CACHE ONLY
   - Leases, heartbeats, transient state
   - If conflict, BEADS WINS
   - Can be rebuilt from Beads + events

## Key Design Patterns

### 1. Event-Driven Architecture (Andon Board)
- In-memory pub/sub event bus
- Non-blocking publish with dead letter queue
- Subscribers react to events asynchronously
- Events logged to SQLite for replay/debugging

**Subscription Matrix:**
| Event | Director | Planner | Engineer | Supervisor | Support |
|-------|----------|---------|----------|------------|---------|
| JobQueued | - | ✓ | - | - | - |
| StationReady | - | ✓ | - | - | - |
| StepQueued | - | - | ✓ | - | - |
| StepDone | ✓ | - | ✓ | - | - |
| StepFailed | ✓ | - | ✓ | ✓ | - |
| OperatorStuck | - | - | - | ✓ | ✓ |

### 2. Single Authority Pattern (Control Room)
- **Plant Director**: Only component that can make policy decisions
- **Planner**: Single dispatcher for work assignment (no race conditions)
- **Engineer**: Evaluates DAGs and routes work (cannot dispatch)

### 3. Lease-Based Crash Recovery
```
1. CLAIMING: Station requests lease on Bead
   SQLite: INSERT INTO leases (...)
   
2. EXECUTION: Operator sends heartbeat every N seconds
   SQLite: UPDATE leases SET expires_at=...
   
3. COMPLETION: Release lease, update Beads
   
4. RECOVERY: On startup, find expired leases
   Check Beads for actual status → Trust Beads
```

### 4. Station Execution Lifecycle
- Each station: Separate git worktree + tmux pane
- Operators spawn at stations and execute work
- Heartbeats track operator health
- Stuck detection triggers recovery

### 5. DAG Workflow Engine
- SOPs defined as Directed Acyclic Graphs
- Steps have dependencies on other steps
- Engine evaluates which steps are ready to run
- Supports parallel execution of independent steps

## Component Boundaries

### Support Service (Read-Only Observer)
**CAN:**
- Subscribe to ALL events
- Emit: Stuck, HealthOK, CleanupDone
- Nudge operators
- Clean up dead stations

**CANNOT:**
- Dispatch work
- Modify job status
- Override planner decisions
- Create/modify travelers

### Director Authority
**CAN:**
- Override planner decisions
- Pause/resume factory
- Escalate issues to humans
- Make policy decisions

**CANNOT:**
- Execute work directly
- Access stations directly
- Modify beads directly

## Data Flow Patterns

### Task Reception Flow
```
User → Director.ReceiveTask() → Create Bead → Create Batch
                                    ↓
                              Planner.Enqueue()
```

### Work Dispatch Flow
```
Planner.ProcessQueue() → Find Available Station
                              ↓
                        Traveler.Attach()
                              ↓
                        Operator.Spawn()
                              ↓
                        Emit(StepStarted)
```

### Merge Flow
```
Merge.Submit() → CheckConflicts()
                      ↓
              ┌───────┴───────┐
              ↓               ↓
          Conflicts      No Conflicts
              ↓               ↓
         Escalate      ProcessQueue()
                              ↓
                         Merge()
```

## Manufacturing Terminology Mapping

| Factory Concept | FactoryAI Implementation |
|-----------------|--------------------------|
| Station | Git worktree + tmux pane |
| Operator | AI agent (Claude) |
| Traveler | Work order document |
| SOP | DAG workflow |
| Formula | TOML recipe |
| Bead | Work item |
| Batch | Group of related work |
| Work Cell | Parallel execution group |
| Andon Board | Event bus |