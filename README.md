# FactoryAI

A manufacturing-plant-inspired multi-agent workspace manager in Go. FactoryAI orchestrates AI agents (via `claude -p` subprocesses) using factory metaphors: parallel Assembly Lines of Stations, with a live Bubbletea TUI.

Inspired by Steve Yegge's [Gas Town](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04).

## Features

- **Factory Metaphor**: Workflows organized as Assembly Lines containing Stations
- **Parallel Execution**: Multiple assembly lines run concurrently
- **Quality Inspection**: Optional validation with retry loops
- **Multiple Merger Strategies**: Concat, Claude-based, or First-success
- **Live TUI**: Real-time progress visualization with Bubbletea
- **YAML Blueprints**: Declarative workflow configuration

## Installation

```bash
go build -o factory ./cmd/factory/main.go
```

## Requirements

- Go 1.22+
- [Claude CLI](https://claude.ai/code) installed and in PATH

## Usage

### List Available Blueprints

```bash
./factory list-blueprints
./factory list-blueprints --dir ./blueprints
```

### Run a Factory

```bash
# With TUI (default)
./factory run --blueprint ./blueprints/research_factory.yaml \
  --task "Explain quantum computing"

# Without TUI (for CI/scripting)
./factory run --blueprint ./blueprints/coding_factory.yaml \
  --task "Design a REST API" --no-tui
```

### Environment Variables

```bash
# Optional: Override claude binary path
export CLAUDE_BIN=/path/to/claude
```

## Architecture

```
Blueprint (YAML config)
    └── Factory (orchestrator)
            └── Assembly Lines (parallel execution)
                    └── Stations (sequential execution)
                            └── Worker + optional Inspector
                                    └── Agent (claude -p subprocess)
```

### Data Flow

1. CLI loads blueprint YAML and creates event channel
2. Factory creates Job (UUID), spins goroutines per AssemblyLine
3. Each AssemblyLine runs its Stations sequentially
4. Each Station: Worker runs agent → Inspector validates → retry on FAIL
5. Station output becomes `{context}` for next station
6. Merger combines outputs from all parallel lines
7. TUI displays progress via events

### Key Components

| Package | Description |
|---------|-------------|
| `internal/config` | YAML blueprint loading |
| `internal/agents` | Agent interface and Claude implementation |
| `internal/worker` | Prompt template rendering |
| `internal/inspector` | Output validation (PASS/FAIL) |
| `internal/station` | Worker + Inspector with retry loop |
| `internal/assemblyline` | Sequential station execution |
| `internal/merger` | Output merging strategies |
| `internal/factory` | Top-level orchestrator |
| `internal/tui` | Bubbletea terminal UI |

## Blueprint Format

```yaml
factory:
  name: "My Factory"
  description: "Description of what this factory does"

  assembly_lines:
    - name: "line-1"
      stations:
        - name: "step-1"
          role: "Role description"
          prompt: |
            Task: {task}
            Context: {context}
          inspector:
            enabled: true
            max_retries: 2
            criteria: "Validation criteria"

    - name: "line-2"
      stations:
        - name: "step-1"
          role: "Another role"
          prompt: "Prompt template with {task}"

  merger:
    type: "concat"  # or "claude" or "first"
    separator: "\n\n---\n\n"
```

### Template Variables

- `{task}` - The original user task
- `{context}` - Previous station's output (empty for first station)
- `{role}` - The station's role name

### Merger Types

- **concat**: Joins outputs with a separator
- **claude**: Uses Claude to intelligently merge outputs
- **first**: Returns the first successful line's output

## Example Blueprints

The `blueprints/` directory includes three example factories:

- **research_factory.yaml**: Parallel literature review and critical analysis
- **coding_factory.yaml**: Parallel implementation and code review
- **review_factory.yaml**: Multi-perspective code review (correctness, style, security)

## License

MIT
