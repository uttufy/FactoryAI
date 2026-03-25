# FactoryAI CLI Reference

Complete reference for all FactoryAI CLI commands.

## Installation

```bash
go build -o factory ./cmd/factory/main.go
sudo mv factory /usr/local/bin/
```

## Global Options

```bash
factory [global-options] <command> [command-options]

Global Options:
  --config string      Path to factory config (default "./configs/factory.yaml")
  --project-path string Path to project directory (default ".")
  -h, --help          Help for factory
  -v, --version       Version number
```

## Commands

### Factory Management

#### `init`

Initialize a new factory.

```bash
factory init [flags]

Flags:
  --project-path string   Path to project directory (default ".")
  --name string           Factory name (default "My Factory")
  --description string    Factory description
```

**Examples:**

```bash
# Initialize in current directory
factory init

# Initialize with custom name
factory init --name "My Awesome Factory"

# Initialize for specific project
factory init --project-path /path/to/project
```

#### `status`

Show factory status.

```bash
factory status

Output:
- Total stations
- Active stations
- Idle stations
- Active operators
- Stuck operators
- Pending merges
- Last activity
```

#### `boot`

Start all stations and prepare factory for work.

```bash
factory boot [flags]

Flags:
  --stations int        Number of stations to start (default: all)
  --background          Run in background without TUI
```

#### `shutdown`

Gracefully shutdown the factory.

```bash
factory shutdown [flags]

Flags:
  --force              Force immediate shutdown (not recommended)
  --wait-timeout       Max time to wait for graceful shutdown (default: 5m)
```

#### `pause`

Pause factory operations.

```bash
factory pause

Effect:
- Stops accepting new work
- Completes in-progress work
- Keeps operators alive
```

#### `resume`

Resume factory operations.

```bash
factory resume

Effect:
- Accepts new work
- Resumes dispatch
```

---

### Station Management

#### `station add`

Add a new station.

```bash
factory station add [flags]

Flags:
  --name string         Station name (required)
  --git-ref string      Git branch/ref to checkout (default: "main")
  --worktree string     Worktree path (auto-generated if not specified)
```

**Examples:**

```bash
# Add station with defaults
factory station add --name "station-1"

# Add station for specific branch
factory station add --name "feature-station" --git-ref "feature/xyz"
```

#### `station list`

List all stations.

```bash
factory station list [flags]

Flags:
  --status string       Filter by status (active, idle, offline)
  --output format       Output format (table, json) (default: table)
```

**Output:**

```
ID          NAME        STATUS    BRANCH      OPERATOR
station-1   Station 1  Active    main        op-123
station-2   Station 2  Idle      main        -
station-3   Station 3  Offline   develop     -
```

#### `station remove`

Remove a station.

```bash
factory station remove <station-id> [flags]

Flags:
  --force              Remove even if operator present
  --cleanup            Remove worktree and tmux session
```

#### `station status`

Show detailed station status.

```bash
factory station status <station-id>

Output:
- Station details
- Current operator
- Active traveler
- Worktree status
- tmux session info
```

---

### Operator Management

#### `operator spawn`

Spawn a new operator at a station.

```bash
factory operator spawn [flags]

Flags:
  --station string      Station ID (required)
  --role string         Operator role (default: "operator")
  --model string        Claude model to use (default: "claude-sonnet-4-20250514")
```

**Examples:**

```bash
# Spawn operator at station
factory operator spawn --station station-1

# Spawn with specific role
factory operator spawn --station station-2 --role "senior-developer"
```

#### `operator list`

List all operators.

```bash
factory operator list [flags]

Flags:
  --status string       Filter by status (active, idle, stuck)
  --station string      Filter by station
```

**Output:**

```
ID          STATION     ROLE      STATUS      HEARTBEAT
op-123      station-1   Developer Active      5s ago
op-124      station-2   Reviewer  Idle        1m ago
op-125      station-3   DevOps    Stuck       5m ago
```

#### `operator status`

Show detailed operator status.

```bash
factory operator status <operator-id>

Output:
- Operator details
- Station assignment
- Current work
- Heartbeat status
- Work history
```

#### `operator decommission`

Decommission an operator.

```bash
factory operator decommission <operator-id> [flags]

Flags:
  --force              Force decommission even if working
  --handoff-to         Transfer work to another operator
```

---

### Work Cell Management

#### `cell create`

Create a new work cell.

```bash
factory cell create [flags]

Flags:
  --name string         Cell name (required)
  --stations strings    Station IDs to include (required)
  --description string  Cell description
```

**Example:**

```bash
factory cell create --name "review-cell" \
  --stations station-1,station-2,station-3 \
  --description "Parallel code review cell"
```

#### `cell activate`

Activate a work cell for parallel execution.

```bash
factory cell activate <cell-id> [flags]

Flags:
  --bead-id string      Bead ID to execute
  --formula string      Formula to use
```

#### `cell status`

Show work cell status.

```bash
factory cell status <cell-id>

Output:
- Cell configuration
- Station status
- Active execution
- Results
```

#### `cell disperse`

Disperse a work cell after completion.

```bash
factory cell disperse <cell-id> [flags]

Flags:
  --cleanup            Clean up worktrees
```

---

### Job Management

#### `job create`

Create a new job.

```bash
factory job create <task> [flags]

Flags:
  --priority int        Job priority (default: 0)
  --formula string      Formula to apply
  --epac string         Epic ID to attach to
```

**Examples:**

```bash
# Simple job
factory job create "Implement feature X"

# With priority
factory job create "Fix bug Y" --priority 100

# With formula
factory job create "Add API endpoint" --formula ./formulas/feature.toml
```

#### `job list`

List all jobs.

```bash
factory job list [flags]

Flags:
  --status string       Filter by status (queued, active, completed, failed)
  --limit int           Max results (default: 50)
```

**Output:**

```
ID          TASK                  STATUS    CREATED
job-123     Implement feature X   Active    5m ago
job-124     Fix bug Y             Queued    2m ago
job-125     Add API endpoint      Completed 1h ago
```

#### `job show`

Show detailed job information.

```bash
factory job show <job-id>

Output:
- Job details
- Current status
- Station/operator assignment
- Work history
- Results
```

#### `job close`

Close a job.

```bash
factory job close <job-id> [flags]

Flags:
  --reason string       Reason for closing
```

#### `job epic`

Create an epic (parent job).

```bash
factory job epic <epic-name> [flags]

Flags:
  --description string  Epic description
```

#### `job add-child`

Add child job to epic.

```bash
factory job add-child <epic-id> <task> [flags]

Flags:
  --priority int        Child job priority
```

---

### Traveler Management

#### `traveler attach`

Attach a traveler to a station.

```bash
factory traveler attach <bead-id> <station-id> [flags]

Flags:
  --formula string      Formula to use
```

#### `traveler show`

Show traveler status.

```bash
factory traveler show <bead-id>

Output:
- Traveler details
- Current station
- Progress
- Step history
```

#### `traveler clear`

Clear traveler from station.

```bash
factory traveler clear <bead-id>
```

---

### Batch Management

#### `batch create`

Create a new batch.

```bash
factory batch create <batch-name> [flags]

Flags:
  --beads strings      Bead IDs to include (comma-separated)
  --formula string     Formula to apply to all
```

**Example:**

```bash
factory batch create "release-batch" \
  --beads bead-123,bead-124,bead-125 \
  --formula ./formulas/release.toml
```

#### `batch status`

Show batch status.

```bash
factory batch status <batch-id>

Output:
- Batch details
- Bead statuses
- Overall progress
- Failures
```

#### `batch list`

List all batches.

```bash
factory batch list [flags]

Flags:
  --status string       Filter by status
  --limit int           Max results
```

#### `batch dashboard`

Show batch dashboard.

```bash
factory batch dashboard <batch-id>

Displays:
- Real-time progress
- Station status
- Operator status
- Event log
```

---

### Formula Management

#### `formula load`

Load a formula.

```bash
factory formula load [flags]

Flags:
  --path string         Path to formula file (required)
```

**Example:**

```bash
factory formula load --path ./formulas/feature.toml
```

#### `formula list`

List all loaded formulas.

```bash
factory formula list

Output:
ID          NAME                VERSION
formula-1   Feature Implement   1.0
formula-2   Code Review         1.2
formula-3   Release             2.0
```

#### `formula create`

Create a new formula from template.

```bash
factory formula create <name> [flags]

Flags:
  --template string    Template to use (feature, bugfix, release, review)
  --description string Formula description
```

#### `formula status`

Show formula status.

```bash
factory formula status <formula-id>

Output:
- Formula details
- Steps
- Dependencies
- Usage statistics
```

---

### Support Commands

#### `health`

Run health check.

```bash
factory health [flags]

Flags:
  --verbose            Show detailed health info
  --json               Output as JSON
```

**Output:**

```
✓ Database: OK
✓ Tmux: OK (3 sessions)
✓ Beads: OK
✓ Disk Space: 15GB available
⚠ Expired Leases: 2

Overall: Healthy
```

#### `cleanup`

Run cleanup operations.

```bash
factory cleanup [flags]

Flags:
  --dry-run            Show what would be cleaned
  --old-events string  Clean events older than (default: 24h)
```

**Cleans:**
- Dead letter queue
- Old events
- Expired leases
- Completed stations

#### `nudge`

Send nudge to operator.

```bash
factory nudge [flags]

Flags:
  --operator string     Operator ID (or use --all)
  --all                Nudge all operators
  --message string     Nudge message (required)
```

**Examples:**

```bash
# Nudge specific operator
factory nudge --operator op-123 --message "Please continue"

# Nudge all operators
factory nudge --all --message "Factory resuming"
```

---

### Merge Queue Commands

#### `mq list`

List merge requests.

```bash
factory mq list [flags]

Flags:
  --status string       Filter by status (pending, checking, conflicted, merging, complete)
```

**Output:**

```
ID          BEAD        BRANCH           STATUS      PRIORITY
mr-1        bead-123    feature/new      Checking    10
mr-2        bead-124    fix/bug-1        Ready       100
mr-3        bead-125    hotfix/crit      Conflicted  1000
```

#### `mq status`

Show merge request status.

```bash
factory mq status <merge-request-id>

Output:
- MR details
- Conflicts (if any)
- Status history
```

#### `mq escalate`

Escalate merge issue for human attention.

```bash
factory mq escalate <merge-request-id> --reason "<reason>"
```

---

### Mail Commands

#### `mail send`

Send mail to operator.

```bash
factory mail send [flags]

Flags:
  --to string          Target operator ID (required)
  --subject string     Mail subject
  --message string     Mail body (or use --file)
  --file string        Read message from file
```

#### `mail read`

Read mail for operator.

```bash
factory mail read [flags]

Flags:
  --operator string    Operator ID (default: current operator)
  --unread-only        Show only unread messages
```

#### `mail broadcast`

Broadcast mail to all operators.

```bash
factory mail broadcast [flags]

Flags:
  --subject string     Subject (required)
  --message string     Message (required)
  --file string        Read from file
```

---

### Role Commands

#### `role start`

Start a role.

```bash
factory role start <role-name> [flags]

Flags:
  --station string     Station to assign to
```

#### `role stop`

Stop a role.

```bash
factory role stop <role-name>
```

#### `role list`

List all roles.

```bash
factory role list

Output:
- Role names
- Status
- Capabilities
```

---

### v0.x Commands (Backward Compatibility)

#### `run`

Run a v0.x blueprint.

```bash
factory run --blueprint <path> --task <task> [flags]

Flags:
  --blueprint string   Blueprint YAML path (required)
  --task string        Task to execute (required)
  --no-tui             Disable TUI
```

#### `list-blueprints`

List available blueprints.

```bash
factory list-blueprints [flags]

Flags:
  --dir string         Blueprints directory (default: ./blueprints)
```

---

## Output Formats

### Table Output (default)

```bash
factory station list
```

```
ID          NAME        STATUS
station-1   Station 1  Active
station-2   Station 2  Idle
```

### JSON Output

```bash
factory station list --output json
```

```json
{
  "stations": [
    {"id": "station-1", "name": "Station 1", "status": "active"},
    {"id": "station-2", "name": "Station 2", "status": "idle"}
  ]
}
```

### Verbose Output

```bash
factory health --verbose
```

```
✓ Database: OK
  Path: /path/to/db
  Size: 15MB
  Connections: 5/100

✓ Tmux: OK
  Sessions: 3
  Active panes: 12
```

## Environment Variables

```bash
# Override default paths
export FACTORY_CONFIG=/path/to/factory.yaml
export FACTORY_PROJECT=/path/to/project

# Override claude binary
export CLAUDE_BIN=/path/to/claude

# Set log level
export FACTORY_LOG_LEVEL=debug

# Set timeouts
export FACTORY_DEFAULT_TIMEOUT=1h
export FACTORY_HEARTBEAT_INTERVAL=30s
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid usage |
| 3 | Network error |
| 4 | Database error |
| 5 | File not found |
| 6 | Permission denied |
| 7 | Timeout |
| 8 | Validation failed |

## Examples

### Typical Workflow

```bash
# 1. Initialize factory
factory init

# 2. Boot the factory
factory boot

# 3. Create a job
factory job create "Implement feature X" --formula ./formulas/feature.toml

# 4. Check status
factory status

# 5. View progress
factory batch dashboard <batch-id>

# 6. When done, shutdown
factory shutdown
```

### Code Review Workflow

```bash
# Create review job
factory job create "Review PR #123" --formula ./formulas/code-review.toml

# Monitor progress
factory job list --status active

# View results
factory job show <job-id>
```

### Release Workflow

```bash
# Create release batch
factory batch create "v1.2.0" \
  --beads bead-1,bead-2,bead-3 \
  --formula ./formulas/release.toml

# Monitor dashboard
factory batch dashboard <batch-id>

# Check merges
factory mq list

# When done
factory batch status <batch-id>
```

## Troubleshooting

### Command Not Found

```bash
# Verify installation
which factory

# Check PATH
echo $PATH | tr ':' '\n' | grep factory
```

### Permission Denied

```bash
# Make executable
chmod +x factory

# Or install to /usr/local/bin
sudo mv factory /usr/local/bin/
```

### Config Not Found

```bash
# Check default location
ls -la ./configs/factory.yaml

# Or specify explicitly
factory --config /path/to/factory.yaml <command>
```

### Station/Operator Issues

```bash
# Check status
factory station status <station-id>
factory operator status <operator-id>

# Run health check
factory health --verbose

# Nudge if stuck
factory nudge --operator <operator-id> --message "Wake up"
```

## See Also

- [Architecture Documentation](ARCHITECTURE.md)
- [Formulas Guide](FORMULAS.md)
- [Events Reference](EVENTS.md)
