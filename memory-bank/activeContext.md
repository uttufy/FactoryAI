# Active Context: FactoryAI

## Current State
The project is in early development with core business logic implemented but no user-facing interface.

## What's Working
- **16 internal packages** fully implemented with business logic
- **Event bus** with pub/sub, dead letter queue
- **SQLite store** with migrations, WAL mode, crash recovery
- **Beads client** for work item management
- **DAG workflow engine** with dependency evaluation
- **Formula parsing** for TOML-based workflows
- **Station manager** with git worktree provisioning
- **Operator pool** with heartbeats and stuck detection
- **All supporting services**: planner, director, supervisor, support, assembly, batch, workcell, mail, tmux

## What's Not Working / Not Implemented
- **CLI entry point** - No `cmd/factory/main.go`
- **TUI dashboard** - Only placeholder exists
- **Configuration loading** - No `factory.yaml` support
- **Example formulas** - No TOML workflow examples
- **Role configurations** - No `configs/roles/`

## Current Focus Areas
1. Need to create CLI entry point (`cmd/factory/main.go`)
2. Need to implement Cobra commands for factory management
3. Need to wire components together in main
4. Need to implement TUI dashboard

## Recent Decisions
- Using collapsed MVP v1 roles (3 roles instead of 8)
- SQLite as runtime cache, Beads as source of truth
- Event-driven architecture for component communication
- Lease-based crash recovery

## Important Context
- Project uses manufacturing terminology throughout
- Beads CLI is external dependency (github.com/steveyegge/beads)
- Claude Code is the AI agent backend
- tmux is required for station UI

## Next Steps
1. Create `cmd/factory/main.go` with Cobra CLI
2. Implement core commands: `init`, `boot`, `status`, `shutdown`
3. Implement station commands: `station add/list/remove`
4. Implement job commands: `job create/list/close`
5. Wire together Director, Planner, StationManager