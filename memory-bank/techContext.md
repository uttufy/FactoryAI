# Technical Context: FactoryAI

## Technology Stack

### Language & Runtime
- **Go 1.24.2** - Primary language
- Module: `github.com/uttufy/FactoryAI`

### Core Dependencies

#### CLI & UI
| Package | Version | Purpose |
|---------|---------|---------|
| `spf13/cobra` | v1.8.1 | CLI framework |
| `charmbracelet/bubbletea` | v1.3.10 | TUI framework |
| `charmbracelet/bubbles` | v1.0.0 | TUI components |
| `charmbracelet/lipgloss` | v1.1.0 | TUI styling |

#### Data & Config
| Package | Version | Purpose |
|---------|---------|---------|
| `mattn/go-sqlite3` | v1.14.22 | SQLite driver |
| `pelletier/go-toml/v2` | v2.2.0 | TOML parsing |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |

#### Utilities
| Package | Version | Purpose |
|---------|---------|---------|
| `google/uuid` | v1.6.0 | UUID generation |
| `golang.org/x/sync` | v0.7.0 | Synchronization primitives |
| `joho/godotenv` | v1.5.1 | Environment variable loading |

## Project Structure

```
FactoryAI/
├── go.mod
├── go.sum
├── plan.md                    # Low-Level Design document
├── README.md
├── .env.example
├── docs/
│   └── ARCHITECTURE.md        # Architecture documentation
├── internal/
│   ├── events/                # Event bus (Andon Board)
│   │   └── bus.go
│   ├── store/                 # SQLite runtime state
│   │   ├── store.go
│   │   └── migrations/
│   │       └── 001_init.sql
│   ├── beads/                 # Beads CLI client
│   │   ├── client.go
│   │   └── types.go
│   ├── workflow/              # DAG engine & formulas
│   │   ├── dag.go
│   │   └── formula.go
│   ├── station/               # Station manager
│   │   └── manager.go
│   ├── operator/              # Operator pool
│   │   └── pool.go
│   ├── traveler/              # Traveler manager
│   │   └── traveler.go
│   ├── planner/               # Production planner
│   │   └── planner.go
│   ├── director/              # Plant director
│   │   └── director.go
│   ├── supervisor/            # Floor supervisor
│   │   └── supervisor.go
│   ├── support/               # Support service
│   │   └── service.go
│   ├── assembly/              # Merge queue
│   │   └── assembly.go
│   ├── batch/                 # Batch manager
│   │   └── batch.go
│   ├── workcell/              # Work cell manager
│   │   └── workcell.go
│   ├── mail/                  # Inter-agent messaging
│   │   └── mail.go
│   ├── tmux/                  # tmux manager
│   │   └── tmux.go
│   └── tui/                   # TUI dashboard (placeholder)
│       └── model.go
└── memory-bank/               # Project memory
```

## Database Schema (SQLite)

### Tables
- `stations` - Station state
- `operators` - Operator state
- `leases` - Crash recovery leases
- `sops` - SOP definitions
- `travelers` - Work orders
- `event_log` - Event history
- `dead_letter` - Dropped events
- `batches` - Production batches
- `merge_requests` - Merge queue
- `factory_status` - Factory state (singleton)

## External Dependencies

### Required Tools
1. **beads CLI** - github.com/steveyegge/beads
   - Work item management
   - Git-backed storage
   
2. **tmux** - Session management
   - Required for station UI
   
3. **claude CLI** - Anthropic's Claude Code
   - AI agent execution
   
4. **Git** - Version control
   - Worktree management

## Build & Run

```bash
# Build
go build ./...

# Test
go test ./...

# Run (when CLI implemented)
./factory init
./factory boot
./factory status
```

## Configuration

### Environment Variables
- `CLAUDE_BIN` - Path to claude binary (optional)

### Config Files (Not Yet Implemented)
- `factory.yaml` - Factory configuration
- `configs/roles/*.yaml` - Role definitions
- `formulas/*.toml` - Workflow recipes