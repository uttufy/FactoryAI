# FactoryAI Quick Start Guide

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.22+** - [Download Go](https://golang.org/dl/)
- **Claude CLI** - [Claude Code](https://claude.ai/code)
- **Beads CLI** - [github.com/steveyegge/beads](https://github.com/steveyegge/beads)
- **tmux** - Terminal multiplexer for session management
- **Git** - Version control for worktrees

## Installation

```bash
# Clone the repository
git clone https://github.com/uttufy/FactoryAI.git
cd FactoryAI

# Build the binary
go build -o factory ./cmd/factory/main.go

# (Optional) Install to PATH
sudo mv factory /usr/local/bin/
```

## Your First Factory

### Step 1: Initialize

Create a factory in your project directory:

```bash
cd your-project
factory init
```

This creates:
- `.factory/` directory
- SQLite database at `.factory/factory.db`
- Default configuration

### Step 2: Boot the Factory

Start all services:

```bash
factory boot
```

This initializes:
- Event bus (Andon Board)
- Station manager
- Operator pool
- DAG workflow engine
- Planner
- Supervisor
- Support service
- Assembly (merge queue)
- Mail system
- Director

### Step 3: Check Status

Verify everything is running:

```bash
factory status
```

Expected output:
```
Factory Status: Running
Uptime: 10s
Active Jobs: 0
Pending Batches: 0

Stations:
  (none)
```

## Working with Stations

### Add a Station

```bash
factory station add --name "dev-station"
```

This creates:
- Git worktree at `.factory/worktrees/dev-station/`
- tmux session `factory-dev-station`

### List Stations

```bash
factory station list
```

### Spawn an Operator

```bash
factory operator spawn --station <station-id>
```

## Working with Jobs

### Create a Job

```bash
factory job create "Implement user authentication"
```

### List Jobs

```bash
factory job list
```

### Show Job Details

```bash
factory job show <job-id>
```

### Close a Job

```bash
factory job close <job-id>
```

## Working with Batches

### Create a Batch

```bash
factory batch create "auth-feature" job-1 job-2 job-3
```

### Track Progress

```bash
factory batch status <batch-id>
```

### View Dashboard

```bash
factory batch dashboard
```

## Working with Formulas

### List Available Formulas

```bash
factory formula list
```

### Show a Formula

```bash
factory formula show feature
```

### Create a Custom Formula

```bash
factory formula create my-workflow
```

### Validate a Formula

```bash
factory formula validate formulas/my-workflow.toml
```

## Running Work

### Execute a Job Immediately

```bash
factory run <job-id>
```

### Dispatch to Specific Station

```bash
factory dispatch <job-id> <station-id>
```

### Generate a Plan from Goal

```bash
factory plan "Build a REST API for user management"
```

This creates:
- A task bead
- An SOP with steps (analyze, plan, implement, review)

## Support Services

### Health Check

```bash
factory support status
```

Output shows:
- Database status
- tmux status
- Beads client status
- Disk space
- Active stations
- Expired leases

### View Logs

```bash
factory support logs
```

### Attach Support to Station

```bash
factory support attach <station-id>
```

## Merge Queue

### Check Queue Status

```bash
factory merge status
```

### List Pending Merges

```bash
factory merge list
```

### Approve a Merge

```bash
factory merge approve <mr-id>
```

### Block a Merge

```bash
factory merge block <mr-id> "Conflicts need resolution"
```

## Communication

### Send Mail to Operator

```bash
factory mail send <operator-id> "Update" "Please review the latest changes"
```

### Broadcast to All

```bash
factory mail broadcast "Factory Notice" "Lunch break in 30 minutes"
```

### List Messages

```bash
factory mail list
```

## Role Management

### List Available Roles

```bash
factory role list
```

Output:
- Built-in roles (developer, architect, reviewer, tester)
- Custom roles from `configs/roles/`

### Set Current Role

```bash
factory role set developer
```

### Clear Role

```bash
factory role clear
```

## Shutdown

### Graceful Shutdown

```bash
factory shutdown
```

This:
- Stops accepting new work
- Completes in-progress work
- Saves state to database
- Cleans up resources

### Pause and Resume

```bash
# Pause operations
factory pause

# Resume operations
factory resume
```

## Complete Example

Here's a complete workflow:

```bash
# 1. Initialize
factory init

# 2. Boot
factory boot &

# 3. Wait for boot
sleep 2

# 4. Add stations
factory station add --name "dev-1"
factory station add --name "dev-2"

# 5. Spawn operators
factory operator spawn --station station-1
factory operator spawn --station station-2

# 6. Create jobs
factory job create "Implement login"
factory job create "Implement logout"

# 7. Create batch
factory batch create "auth" job-1 job-2

# 8. Check status
factory status

# 9. View dashboard
factory batch dashboard

# 10. When done, shutdown
factory shutdown
```

## Troubleshooting

### Factory Not Initialized

```
Error: factory not initialized. Run 'factory init' first
```

Solution: Run `factory init`

### Factory Not Booted

```
Error: factory not booted. Run 'factory boot' first
```

Solution: Run `factory boot`

### Station Not Found

```
Error: station not found: station-1
```

Solution: Check station ID with `factory station list`

### Operator Not Responding

```
Error: operator stuck
```

Solution: Check with `factory operator status <id>` or nudge with `factory nudge --operator <id> --message "Please continue"`

## Next Steps

- Read [CLI.md](./CLI.md) for complete command reference
- Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system design
- Read [FORMULAS.md](./FORMULAS.md) for workflow recipes
- Read [EVENTS.md](./EVENTS.md) for event system
