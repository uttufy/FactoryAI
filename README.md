# FactoryAI

A multi-agent workspace manager that orchestrates parallel AI agents working on software development tasks using manufacturing factory concepts.

## Overview

FactoryAI uses real manufacturing terminology adapted for software production:

| Factory Concept | Software Analog | Description |
|-----------------|-----------------|-------------|
| **Station** | Workspace | Isolated git worktree for work |
| **Operator** | AI Agent | Claude agent working at a station |
| **Traveler** | Work Order | Tracks work through stations |
| **SOP** | Workflow | Standard Operating Procedure (DAG) |
| **Formula** | Recipe | TOML-based workflow template |
| **Bead** | Work Item | Unit of work managed by beads CLI |
| **Batch** | Production Batch | Group of related work items |

## Prerequisites

- **Go 1.22+** - For building the CLI
- **tmux** - Required for session management
- **Git** - Required for worktree management
- **beads CLI** (optional) - For full work item management
- **claude CLI** (optional) - For AI agent execution

## Installation

```bash
# Clone the repository
git clone https://github.com/uttufy/FactoryAI.git
cd FactoryAI

# Build the CLI
go build -o factory ./cmd/factory

# Optionally, move to PATH
sudo mv factory /usr/local/bin/
```

## Quick Start

### 1. Initialize a Factory

```bash
# In your project directory (must be a git repository)
factory init
```

This creates:
- `.factory/` directory for state storage
- `factory.yaml` configuration file
- SQLite database

### 2. Check Factory Status

```bash
factory status
```

### 3. Add a Station

```bash
factory station add station-1
```

This creates an isolated git worktree for parallel work.

### 4. Create Jobs

```bash
# Create a job
factory job create "Implement user authentication"

# List jobs
factory job list
```

### 5. Run a Formula (Workflow)

```bash
# List available formulas
factory formula list

# Show formula details
factory formula show formulas/feature.toml

# Run a formula with a task
factory run --formula formulas/feature.toml --task "Build REST API"
```

### 6. Manage Batches

```bash
# Create a batch
factory batch create "Auth Feature" job-1 job-2 job-3

# List batches
factory batch list

# Show batch status
factory batch status batch-1
```

## CLI Commands

### Factory Management

```bash
factory init                        # Initialize a new factory
factory boot                        # Start the factory (background services)
factory status                      # Show factory status
factory shutdown                    # Graceful shutdown
factory pause                       # Pause factory operations
factory resume                      # Resume factory operations
```

### Station Management

```bash
factory station add <name>          # Create a new station
factory station list                # List all stations
factory station remove <id>         # Remove a station
factory station status <id>         # Show station details
```

### Job Management

```bash
factory job create <title>          # Create a new job
factory job list                    # List all jobs
factory job show <id>               # Show job details
factory job close <id>              # Close a job
factory job dispatch <job> <station> # Dispatch job to station
```

### Traveler Management

```bash
factory traveler attach <station> <job>  # Attach work to station
factory traveler show <station>          # Show station's traveler
factory traveler clear <station>         # Clear station's traveler
```

### Batch Management

```bash
factory batch create <name> <jobs...>    # Create a batch
factory batch list                       # List all batches
factory batch status <id>                # Show batch status
```

### Formula Management

```bash
factory formula list                       # List available formulas
factory formula show <path>                # Show formula details
factory formula cook <path>                # Cook formula into SOP
factory run --formula <path> --task "<task>" # Run a formula
```

## Formulas

Formulas are TOML-based workflow recipes. Example (`formulas/feature.toml`):

```toml
name = "feature-implementation"
description = "Standard workflow for implementing a new feature"

[variables]
task = ""

[[steps]]
name = "analyze"
description = "Analyze the task requirements"
acceptance = "Clear understanding of what needs to be implemented"

[[steps]]
name = "design"
description = "Design the solution architecture"
dependencies = ["analyze"]
acceptance = "Design document is ready"

[[steps]]
name = "implement"
description = "Implement the feature"
dependencies = ["design"]
acceptance = "Feature code is written"

[[steps]]
name = "test"
description = "Write and run tests"
dependencies = ["implement"]
acceptance = "All tests pass"

[[steps]]
name = "review"
description = "Review the implementation"
dependencies = ["test"]
acceptance = "Code review completed"
```

### Included Formulas

- `formulas/feature.toml` - Feature implementation workflow
- `formulas/bugfix.toml` - Bug fix workflow
- `formulas/code-review.toml` - Code review workflow

## Architecture

FactoryAI uses a four-layer architecture:

1. **Interface Layer** - CLI (Cobra), TUI (Bubble Tea), HTTP Server
2. **Control Room** - Plant Director, Planner, Process Engineer, Event Bus
3. **Production Floor** - Stations, Operators, Supervisors, Assembly
4. **Data Plane** - Beads (Git-backed), SQLite (runtime cache)

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Configuration

Configuration is stored in `factory.yaml`:

```yaml
factory:
  name: "FactoryAI"
  project_path: "."
  max_stations: 10

database:
  path: ".factory/factory.db"

beads:
  binary_path: "beads"

heartbeat:
  interval: 30s
  ttl: 5m

lease:
  ttl: 10m
```

## Development

```bash
# Build
go build ./...

# Test
go test ./...

# Run linter
golangci-lint run
```

## License

MIT License - See [LICENSE](LICENSE) for details.