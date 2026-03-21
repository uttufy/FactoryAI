# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

FactoryAI is a manufacturing-plant-inspired multi-agent workspace manager in Go. It orchestrates AI agents (via `claude -p` subprocesses) using factory metaphors: parallel Assembly Lines of Stations, with a live Bubbletea TUI.

## Build & Verify Commands

```bash
go build ./...          # Compile all packages
go vet ./...            # Static analysis
go build -o factory ./cmd/factory/main.go   # Build binary
```

## CLI Usage

```bash
# List available blueprints
./factory list-blueprints [--dir ./blueprints]

# Run a factory with TUI
./factory run --blueprint ./blueprints/research_factory.yaml --task "Your task here"

# Run without TUI (for testing/CI)
./factory run --blueprint ./blueprints/research_factory.yaml --task "Your task" --no-tui
```

## Architecture

The system follows a hierarchical factory model:

```
Blueprint (YAML config)
    └── Factory (orchestrator)
            └── Assembly Lines (parallel execution via errgroup)
                    └── Stations (sequential execution)
                            └── Worker + optional Inspector
                                    └── Agent (claude -p subprocess)
```

### Key Packages

- `internal/config` - YAML blueprint loading (`Blueprint`, `FactoryConfig`, `AssemblyLineConfig`, `StationConfig`)
- `internal/job` - Data types (`Job`, `StationResult`, `LineResult`, `JobResult`)
- `internal/agents` - Agent interface and `ClaudeAgent` implementation (wraps `claude -p`)
- `internal/worker` - Wraps Agent + StationConfig, renders prompt templates
- `internal/inspector` - Validates station output, parses PASS/FAIL
- `internal/station` - Composes Worker + Inspector, handles retry loop
- `internal/assemblyline` - Runs stations sequentially, passes context between them
- `internal/merger` - Merges outputs from parallel lines (Concat, Claude, First strategies)
- `internal/factory` - Top-level orchestrator, spins goroutines per assembly line
- `internal/tui` - Bubbletea TUI with lipgloss styling
- `cmd/factory` - Cobra CLI entrypoint

### Data Flow

1. CLI loads blueprint YAML and creates event channel
2. Factory creates Job with UUID, spins goroutines per AssemblyLine
3. Each AssemblyLine runs its Stations sequentially
4. Each Station: Worker runs agent → Inspector validates → retry on FAIL
5. Station output becomes `{context}` for next station in line
6. Merger combines outputs from all lines
7. TUI displays progress via events (`EvtStationStarted`, `EvtStationDone`, etc.)

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/joho/godotenv` - .env loading
- `github.com/google/uuid` - UUID generation
- `golang.org/x/sync` - errgroup for parallel execution

## Requirements

- The `claude` binary must be in PATH (or set via `CLAUDE_BIN` env var)
- Go 1.22+
