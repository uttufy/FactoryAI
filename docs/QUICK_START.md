# FactoryAI Quick Start Guide

> **Important Note:** FactoryAI v1.0 architecture is implemented but most commands are **TODO stubs**. The **v0.x blueprint system** (`factory run`) is **fully functional** and ready to use.

## What Works Right Now

| Command | Status | Description |
|---------|--------|-------------|
| `factory run` | ✅ **Working** | Execute blueprint with task |
| `factory list-blueprints` | ✅ **Working** | List available blueprints |
| `factory init` | 🚧 Stub | Just prints message |
| `factory boot` | 🚧 Stub | Just prints message |
| `factory status` | 🚧 Stub | Just prints message |
| `factory job create` | 🚧 Stub | Just prints message |
| `factory job list` | 🚧 Stub | Shows empty list |
| Other v1.0 commands | 🚧 Stub | Not implemented yet |

---

## Quick Start (Working Method)

### Prerequisites

```bash
# Check Go version (need 1.22+)
go version

# Check Claude CLI (required)
which claude

# Check tmux (required)
which tmux
```

### Step 1: Build

```bash
cd /Users/utkarshsharma/projects/FactoryAI
go build -o factory ./cmd/factory/main.go
```

### Step 2: Add to PATH

```bash
# Add to shell
echo 'export PATH="$PATH:/Users/utkarshsharma/projects/FactoryAI"' >> ~/.zshrc
source ~/.zshrc

# Verify
factory --help
```

### Step 3: Run a Task

```bash
# Navigate to your project
cd ~/projects/test

# Run with blueprint (using full path)
factory run \
  --blueprint /Users/utkarshsharma/projects/FactoryAI/blueprints/research_factory.yaml \
  --task "Explain what a factory pattern is in software engineering"

# Or without TUI (faster, no UI)
factory run \
  --blueprint /Users/utkarshsharma/projects/FactoryAI/blueprints/research_factory.yaml \
  --task "Explain the difference between TCP and UDP" \
  --no-tui
```

---

## Using Available Blueprints

### List Blueprints

```bash
cd /Users/utkarshsharma/projects/FactoryAI
factory list-blueprints
```

### Research Factory

For research and explanation tasks:

```bash
factory run \
  --blueprint ./blueprints/research_factory.yaml \
  --task "Explain quantum computing to a 10-year-old" \
  --no-tui
```

### Coding Factory

For code generation tasks:

```bash
factory run \
  --blueprint ./blueprints/coding_factory.yaml \
  --task "Write a Python function to calculate fibonacci numbers" \
  --no-tui
```

### Review Factory

For code review tasks:

```bash
factory run \
  --blueprint ./blueprints/review_factory.yaml \
  --task "Review this code: \nfunc add(a, b int) int { return a + b }" \
  --no-tui
```

---

## Creating Your Own Blueprint

Create `my-blueprint.yaml`:

```yaml
factory:
  name: "My Factory"
  description: "My custom factory"

  assembly_lines:
    - name: "main-line"
      stations:
        - name: "task"
          role: "Assistant"
          prompt: |
            Task: {task}

            Please provide a clear, well-structured response.
```

Run it:

```bash
factory run --blueprint ./my-blueprint.yaml --task "Explain Big O notation"
```

---

## Blueprint Structure

```yaml
factory:
  name: "Factory Name"
  description: "Description"

  assembly_lines:           # Parallel execution tracks
    - name: "line-1"
      stations:
        - name: "step-1"
          role: "Developer"
          prompt: "Task: {task}\nContext: {context}"
          inspector:        # Optional quality check
            enabled: true
            criteria: "Must be correct"

    - name: "line-2"        # Runs in parallel with line-1
      stations:
        - name: "alternative"
          role: "Architect"
          prompt: "Alternative approach for: {task}"

  merger:
    type: "concat"          # How to combine outputs
    separator: "\n\n---\n\n"
```

### Template Variables

- `{task}` - Your original task
- `{context}` - Output from previous station
- `{role}` - Current station's role

---

## Sample Tasks to Try

```bash
# Explanations
factory run -b ./blueprints/research_factory.yaml -t "Explain monads" --no-tui

# Code generation
factory run -b ./blueprints/coding_factory.yaml -t "Write a REST API in Go" --no-tui

# Analysis
factory run -b ./blueprints/research_factory.yaml -t "Compare SQL vs NoSQL" --no-tui

# Documentation
factory run -b ./blueprints/research_factory.yaml -t "Document the installation process" --no-tui
```

---

## Troubleshooting

### "blueprint not found"

```bash
# Use absolute path
factory run \
  --blueprint /Users/utkarshsharma/projects/FactoryAI/blueprints/research_factory.yaml \
  --task "your task"
```

### "claude: command not found"

```bash
# Install Claude CLI
# See: https://claude.ai/code

# Or specify path
export CLAUDE_BIN=/path/to/claude
```

### TUI Issues

```bash
# Use --no-tui flag
factory run --blueprint ./blueprint.yaml --task "task" --no-tui
```

---

## Why v1.0 Commands Don't Work Yet

The v1.0 architecture has been **designed and implemented structurally**, but the command implementations are still TODO stubs. For example:

```go
func initializeFactory(cmd *cobra.Command, args []string) error {
    fmt.Println("Initializing factory...")
    // TODO: Create .factory directory, initialize database, etc.
    return nil
}

func bootFactory(cmd *cobra.Command, args []string) error {
    fmt.Println("Booting factory...")
    // TODO: Start all services
    return nil
}
```

**What's been implemented:**
- ✅ Complete package structure (director, planner, supervisor, etc.)
- ✅ Event bus (Andon Board)
- ✅ SQLite store with migrations
- ✅ tmux manager
- ✅ Beads client
- ✅ DAG workflow engine
- ✅ Formulas system
- ✅ All v0.x blueprint execution

**What needs implementation:**
- 🚧 CLI command implementations (connecting to the packages)
- 🚧 Director initialization logic
- 🚧 Station provisioning
- 🚧 Operator spawning
- 🚧 End-to-end workflow execution

---

## Complete Example

```bash
# 1. Build
cd /Users/utkarshsharma/projects/FactoryAI
go build -o factory ./cmd/factory/main.go

# 2. Add to PATH
export PATH="$PATH:/Users/utkarshsharma/projects/FactoryAI"

# 3. Navigate to your project
cd ~/projects/my-project

# 4. Run a task
factory run \
  --blueprint /Users/utkarshsharma/projects/FactoryAI/blueprints/research_factory.yaml \
  --task "Explain microservices architecture with pros and cons" \
  --no-tui

# Output shows:
# - Job ID
# - Assembly line progress
# - Station execution
# - Final merged result
```

---

## Next Steps

1. **Use the working commands** - `factory run` and `factory list-blueprints`
2. **Create custom blueprints** - Design your own workflows
3. **Read the documentation:**
   - [ARCHITECTURE.md](ARCHITECTURE.md) - Complete v1.0 system design
   - [FORMULAS.md](FORMULAS.md) - Formula/recipe system
   - [CLI.md](CLI.md) - Complete command reference
   - [EVENTS.md](EVENTS.md) - Event system

---

## Getting Help

```bash
# General help
factory --help

# Command-specific help
factory run --help
factory list-blueprints --help

# List blueprints
factory list-blueprints
```
