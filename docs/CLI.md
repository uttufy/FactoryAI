# FactoryAI CLI Reference

## Overview

FactoryAI provides 46 CLI commands organized into 14 command groups.

## Command Groups

| Group | Commands |
|-------|----------|
| Factory | `init`, `status`, `boot`, `shutdown`, `pause`, `resume` |
| Station | `station add`, `station list`, `station remove`, `station status` |
| Operator | `operator spawn`, `operator list`, `operator status`, `operator decommission` |
| Work Cell | `cell create`, `cell activate`, `cell status`, `cell disperse` |
| Job | `job create`, `job list`, `job show`, `job close`, `job epic` |
| Traveler | `traveler attach`, `traveler show`, `traveler clear` |
| Batch | `batch create`, `batch status`, `batch list`, `batch dashboard` |
| Formula | `formula create`, `formula list`, `formula show`, `formula validate` |
| SOP | `sop list`, `sop show`, `sop execute` |
| Execution | `run`, `dispatch`, `plan` |
| Support | `support status`, `support logs`, `support attach` |
| Merge | `merge status`, `merge list`, `merge approve`, `merge block` |
| Mail | `mail send`, `mail broadcast`, `mail list` |
| Role | `role list`, `role set`, `role clear` |

## Factory Commands

### `factory init`

Initialize a new factory in the current directory.

```bash
factory init [flags]

Flags:
  --project-path string   Project path (default: ".")
  --config string        Config file path (default: "./configs/factory.yaml")
```

Creates:
- `.factory/` directory
- SQLite database at `.factory/factory.db`
- Default configuration

### `factory boot`

Start the factory and all services.

```bash
factory boot [flags]

Flags:
  --max-stations int   Maximum number of stations (default: 5)
```

Initializes:
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

### `factory status`

Show current factory status.

```bash
factory status
```

Output:
- Running state
- Uptime
- Active jobs count
- Station statuses

### `factory shutdown`

Gracefully shutdown the factory.

```bash
factory shutdown
```

### `factory pause` / `factory resume`

Pause or resume factory operations.

```bash
factory pause
factory resume
```

---

## Station Commands

### `station add`

Add a new station.

```bash
factory station add --name <name> [flags]

Flags:
  --name string         Station name (required)
```

Creates:
- Git worktree
- tmux session

### `station list`

List all stations.

```bash
factory station list
```

### `station status`

Show station details.

```bash
factory station status <station-id>
```

### `station remove`

Remove a station.

```bash
factory station remove <station-id>
```

---

## Operator Commands

### `operator spawn`

Spawn an operator at a station.

```bash
factory operator spawn --station <station-id>
```

### `operator list`

List all operators.

```bash
factory operator list
```

### `operator status`

Show operator details.

```bash
factory operator status <operator-id>
```

### `operator decommission`

Decommission an operator.

```bash
factory operator decommission <operator-id>
```

---

## Work Cell Commands

### `cell create`

Create a work cell for parallel execution.

```bash
factory cell create <name> <station-ids...>
```

### `cell activate`

Activate parallel execution.

```bash
factory cell activate <cell-id>
```

### `cell status`

Show work cell status.

```bash
factory cell status <cell-id>
```

### `cell disperse`

Disperse a work cell.

```bash
factory cell disperse <cell-id>
```

---

## Job Commands

### `job create`

Create a new job.

```bash
factory job create <title>
```

### `job list`

List all jobs.

```bash
factory job list
```

### `job show`

Show job details.

```bash
factory job show <job-id>
```

### `job close`

Close a job.

```bash
factory job close <job-id>
```

---

## Traveler Commands

### `traveler attach`

Attach work to a station.

```bash
factory traveler attach <station-id> <bead-id>
```

### `traveler show`

Show traveler status.

```bash
factory traveler show <station-id>
```

### `traveler clear`

Clear traveler from station.

```bash
factory traveler clear <station-id>
```

---

## Batch Commands

### `batch create`

Create a new batch.

```bash
factory batch create <name> <bead-ids...>
```

### `batch status`

Show batch status.

```bash
factory batch status <batch-id>
```

### `batch list`

List all batches.

```bash
factory batch list
```

### `batch dashboard`

Show batch dashboard (TUI).

```bash
factory batch dashboard
```

---

## Formula Commands

### `formula create`

Create a new formula template.

```bash
factory formula create <name>
```

Creates a TOML file in `formulas/` directory.

### `formula list`

List available formulas.

```bash
factory formula list
```

### `formula show`

Show formula details.

```bash
factory formula show <name>
```

### `formula validate`

Validate a formula file.

```bash
factory formula validate <file>
```

---

## SOP Commands

### `sop list`

List all SOPs.

```bash
factory sop list
```

### `sop show`

Show SOP details.

```bash
factory sop show <sop-id>
```

### `sop execute`

Execute an SOP.

```bash
factory sop execute <sop-id>
```

---

## Execution Commands

### `run`

Run a job immediately.

```bash
factory run <job-id>
```

### `dispatch`

Dispatch a job to a station.

```bash
factory dispatch <job-id> <station-id>
```

### `plan`

Generate a plan from a goal.

```bash
factory plan "<goal>"
```

Creates a task bead and SOP with steps.

---

## Support Commands

### `support status`

Run health check.

```bash
factory support status
```

Output:
- Database status
- tmux status
- Beads client status
- Disk space
- Active stations
- Expired leases

### `support logs`

View event logs.

```bash
factory support logs [type]
```

### `support attach`

Attach support to a station.

```bash
factory support attach <station-id>
```

---

## Merge Commands

### `merge status`

Show merge queue status.

```bash
factory merge status
```

### `merge list`

List pending merges.

```bash
factory merge list
```

### `merge approve`

Approve and execute a merge.

```bash
factory merge approve <mr-id>
```

### `merge block`

Block a merge request.

```bash
factory merge block <mr-id> <reason>
```

---

## Mail Commands

### `mail send`

Send a message.

```bash
factory mail send <to> <subject> <body>
```

### `mail broadcast`

Broadcast to all stations.

```bash
factory mail broadcast <subject> <body>
```

### `mail list`

List messages.

```bash
factory mail list [station-id]
```

---

## Role Commands

### `role list`

List available roles.

```bash
factory role list
```

### `role set`

Set current operator role.

```bash
factory role set <role-name>
```

### `role clear`

Clear current role.

```bash
factory role clear
```

---

## Global Flags

Available on all commands:

```bash
--config string         Config file path (default: "./configs/factory.yaml")
--project-path string   Project path (default: ".")
```

## Environment Variables

```bash
CLAUDE_BIN           Override Claude binary path
FACTORY_CONFIG       Override config file path
FACTORY_PROJECT      Override project path
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Command usage error |
