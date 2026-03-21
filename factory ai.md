# FactoryAI — Implementation Plan

A manufacturing-plant-inspired multi-agent workspace manager in Go. Inspired by Steve Yegge's Gas Town, FactoryAI translates factory metaphors into AI agent orchestration: parallel Assembly Lines of Stations, each powered by a `claude -p` subprocess, with a live Bubbletea TUI.

---

## Proposed Changes

### Project Scaffold

#### [NEW] go.mod
Module `github.com/uttufy/FactoryAI`, Go 1.22, with all listed dependencies.

#### [NEW] .env.example
Documents `CLAUDE_BIN` (path override), `ANTHROPIC_API_KEY`.

---

### Config

#### [NEW] internal/config/config.go
- Loads YAML blueprint via `gopkg.in/yaml.v3`
- Structs: `Blueprint`, `FactoryConfig`, `AssemblyLineConfig`, `StationConfig`, `InspectorConfig`, `MergerConfig`
- `LoadBlueprint(path string) (*Blueprint, error)`

---

### Job Types

#### [NEW] internal/job/job.go
- `Job` — ID (UUID), task, blueprint name, timestamps
- `StationResult` — station name, output, duration, pass/fail, retries used
- `LineResult` — line name, list of StationResults, final output, error
- `JobResult` — JobID, all LineResults, final merged output, total duration

---

### Agents

#### [NEW] internal/agents/agent.go
- `Agent` interface with `Run(ctx, Request) (Response, error)` and `Name() string`
- `Request{SystemPrompt, Task, Context}`, `Response{Output, Agent, DurationSec}`

#### [NEW] internal/agents/claude.go
- `ClaudeAgent` struct with optional binary path (default `"claude"`)
- Builds combined prompt from SystemPrompt + (if Task non-empty) task section + (if Context non-empty) context section
- Runs `exec.CommandContext(ctx, binaryPath, "-p", prompt)`
- Captures combined stdout/stderr, returns `Response`
- Validates `claude` binary is in PATH at construction time

#### [NEW] internal/agents/registry.go
- `NewAgent(agentType, binaryPath string) (Agent, error)`
- Currently supports `"claude"`; returns clear error for unknown types

---

### Worker

#### [NEW] internal/worker/worker.go
- `Worker` wraps an `Agent` and a `StationConfig`
- `Run(ctx, task, context string, events chan<- Event) (StationResult, error)`
- Renders prompt template (`{task}`, `{context}`, `{role}` substitution)
- Sends `EvtStationStarted` before invoking the agent

---

### Inspector

#### [NEW] internal/inspector/inspector.go
- `Inspector` wraps an `Agent` and `InspectorConfig`
- `Inspect(ctx, output string) (passed bool, reasoning string, err error)`
- Builds the exact inspection prompt specified in the spec
- Parses first word of response for `PASS`/`FAIL`

---

### Station

#### [NEW] internal/station/station.go
- `Station` composes `Worker` + optional `Inspector`
- `Run(ctx, task, context string, events chan<- Event) (StationResult, error)`
- Retry loop up to `InspectorConfig.MaxRetries` on FAIL
- Sends `EvtStationInspecting` when validation starts
- Sends `EvtStationDone` or `EvtStationFailed` on completion

---

### Assembly Line

#### [NEW] internal/assemblyline/assemblyline.go
- `AssemblyLine` holds a list of `Station`s
- `Run(ctx, task string, events chan<- Event) (LineResult, error)`
- Runs stations sequentially; passes previous station's output as `{context}`
- If a station fails, returns `LineResult` with error (non-fatal at factory level)

---

### Merger

#### [NEW] internal/merger/merger.go
- `Merger` interface + two implementations:
  - `ConcatMerger` — joins outputs with a separator
  - `ClaudeMerger` — uses a `ClaudeAgent` call with the merger prompt template
  - `FirstMerger` — returns first successful line's output
- `Merge(ctx, task string, results []LineResult) (string, error)`
- If only one line configured, the factory skips the merger and uses its output directly

---

### Factory

#### [NEW] internal/factory/factory.go
- `Factory` holds `Blueprint` + event channel
- `Run(ctx, task string, events chan<- Event) (JobResult, error)`
  1. Creates `Job` with UUID
  2. Spins goroutines via `errgroup` — one per AssemblyLine
  3. Collects `[]LineResult` into pre-allocated slice with line index (no mutex needed)
  4. Calls `merger.Merge()`
  5. Sends `EvtMerging{}` and `EvtDone{output}`
  6. Returns `JobResult`

---

### TUI

#### [NEW] internal/tui/model.go
- `Model` struct: `FactoryName`, `JobID`, `Lines []LineView`, `FinalOutput`, `Done`, `events <-chan Event`
- `LineView{Name, Stations []StationView}`
- `StationView{Name, Status, Duration}`
- `Init()` returns command that listens to the event channel via `tea.Every` / custom cmd

#### [NEW] internal/tui/update.go
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)`
- Handles all `Evt*` message types, updating `LineView`/`StationView` accordingly
- On `EvtDone` sets `Done=true`, `FinalOutput`
- Also handles `tea.KeyMsg` for `q`/`ctrl+c` to quit

#### [NEW] internal/tui/view.go
- `View() string` using lipgloss
- One bordered box per assembly line
- Station rows with status icons: `○` pending, `⠿` running, `🔍` inspecting, `✓` done, `✗` failed
- Duration shown right-aligned per row
- Final merged output shown at the bottom when `Done=true`

---

### CLI

#### [NEW] cmd/factory/main.go
- Cobra root command
- `factory run --blueprint <path> --task "<task>" [--no-tui]`
  - Loads `.env` (ignore error if missing)
  - Loads blueprint from path
  - Creates event channel
  - Starts factory goroutine
  - If `--no-tui`: plain `fmt.Println` progress, waits for `EvtDone`
  - Else: starts Bubbletea program with `tui.NewModel(blueprint, events)`
- `factory list-blueprints [--dir <path>]`
  - Defaults dir to `./blueprints`
  - Lists all `*.yaml` files + their `factory.name`/`description`

---

### Blueprints

#### [NEW] blueprints/research_factory.yaml
Two parallel lines: `literature-review` (search → synthesize with inspector) and `critical-analysis` (critique). Claude merger.

#### [NEW] blueprints/coding_factory.yaml
Two parallel lines: `implementation` (architect → implement → test-design) and `review` (security-review → perf-review). Concat merger.

#### [NEW] blueprints/review_factory.yaml
Three parallel lines: `correctness` (logic-check → edge-cases), `style` (readability → maintainability), `security` (vuln-scan → threat-model). Claude merger.

---

## Verification Plan

### Automated Tests
```bash
cd /Users/utkarshsharma/projects/FactoryAI
go build ./...          # must compile with zero errors
go vet ./...            # must pass with zero warnings
```

### Manual Verification
```bash
# List blueprints
./factory list-blueprints --dir ./blueprints

# Run with --no-tui (requires claude in PATH)
./factory run --blueprint ./blueprints/research_factory.yaml \
  --task "What is quantum computing?" --no-tui

# Build binary
go build -o factory ./cmd/factory/main.go
```

> [!NOTE]
> The `claude` binary must be installed. If not present, the tool should print a clear error and exit non-zero. The `--no-tui` mode lets us validate the full pipeline without needing an interactive terminal.
