Based on my reading of the full [plan.md](cci:7://file:///Users/utkarshsharma/projects/FactoryAI/plan.md:0:0-0:0), here are the **architecture-specific** improvements:

---

## Architecture Improvements

### 1. 🚨 Missing Central Event Bus

**Current:** Components communicate only through polling (patrol loops) and the mail/beads system.

**Problem:** Deacon has to *actively loop* to discover that a Worker died or a Merge failed. This creates latency and unnecessary CPU churn.

**Improvement:** Add an internal event bus (Go channels or a lightweight pub-sub like NATS/Redis Streams):
```
Worker crashes → emits "worker.failed" event
  → Witness reacts immediately
  → Deacon is notified
  → PlantManager logs it
```
All patrol loops then become *event-driven handlers* rather than polling timers.

---

### 2. 🏭 `Factory` Is a Ghost Struct

**Current:** `PlantManager` and `Deacon` both hold `*factory.Factory`, but it's **never defined** in the LLD. It's the central orchestrator but has no spec.

**Improvement:** Define `Factory` explicitly — it should own:
- Registry of all Production Lines
- Registry of all running agents (by role)
- The event bus
- Global state (running/paused/shutdown)

---

### 3. 🔀 Serial Merge Queue Blocks Everything

**Current:** Each Production Line has a single `Current *MergeRequest` — one merge at a time, serial.

**Problem:** If a merge takes 5 minutes (CI pipeline), all other workers on that line are blocked from landing their work.

**Improvement:**
- Allow merges from **independent branches** to process concurrently
- Pre-check for file-level conflicts before enqueuing (conflict detection layer)
- Priority queue (not just `[]*MergeRequest`) so urgent MRs jump the line

---

### 4. ⚗️ Workflow Engine Has No Parallel Step Execution

**Current:** `Molecule.CurrentStep int` implies strictly sequential steps. Yet `Epic.Parallel` and Swarms exist to do parallel work.

**Problem:** You can parallelize *workers* but not *steps within a workflow* — the two parallelism models are misaligned.

**Improvement:** Change the step model to a **DAG** (Directed Acyclic Graph):
```
Step A ──→ Step C ──→ Step E
Step B ──/           ↑
Step D ──────────────/
```
Steps with satisfied dependencies run concurrently. `CurrentStep int` → `ReadySteps []*Step`.

---

### 5. 🏊 No Autoscaling of Worker Pools

**Current:** `WorkerPool` takes a fixed `maxWorkers int` at construction.

**Problem:** A quiet production line wastes resources; a flooded one can't burst.

**Improvement:** Add an autoscaling policy per pool:
```go
type ScalingPolicy struct {
    MinWorkers int
    MaxWorkers int
    ScaleUpThreshold   int // queue depth to trigger scale-up
    ScaleDownAfter     time.Duration
}
```
Deacon or Witness can trigger scale decisions based on queue depth and idle time.

---

### 6. 🤝 No Agent Interface Definition

**Current:** `agents.Agent` is used everywhere (Refinery, Witness, Deacon, Dogs) but is never specified. It's the most-used interface in the system.

**Improvement:** Define it explicitly:
```go
type Agent interface {
    ID() string
    Role() string
    SendPrompt(ctx context.Context, prompt string) (string, error)
    GetSessionID() string
    IsAlive() bool
}
```
This also enables a **mock agent** for testing without spinning up real Claude sessions.

---

### 7. 🕳️ No Crash Recovery / Durable Execution

**Current:** NDI says "workflows are durable, survive crashes." But there's no mechanism described.

**Problem:** If a Worker crashes mid-step, the Step is `StepRunning` forever and no one reclaims it.

**Improvement:** Add a `lease` / heartbeat model:
- Each running step has a `Lease` (expires in N seconds)
- Witness checks for expired leases and resets them to `pending`
- The molecule re-runs the step from scratch (idempotency is already a design goal)

---

### 8. 🌐 No Observability Layer

**Current:** Factory status is only queryable via `factory status` CLI.

**Problem:** No way for external systems, CI/CD, or dashboards to observe factory health.

**Improvement:** Expose a lightweight HTTP server from the factory process:
- `GET /health` — liveness/readiness
- `GET /metrics` — Prometheus-compatible (queue depths, worker counts, merge rates)
- `GET /status` — JSON factory snapshot

This is especially important for the "Federation" future goal (you'll need this to connect remote factories).

---

### Summary Table

| Area | Current State | Recommended Fix |
|------|--------------|-----------------|
| Communication | Polling-based patrol loops | Event bus (pub-sub) |
| `Factory` struct | Undefined | Define with line registry + event bus |
| Merge Queue | Serial, single current MR | Parallel + conflict pre-check |
| Workflow steps | Sequential (`CurrentStep int`) | DAG-based execution |
| Worker Pool | Fixed `maxWorkers` | Autoscaling policy |
| Agent interface | Undefined | Explicit interface with `IsAlive`, `SendPrompt` |
| Crash recovery | "Guaranteed" but no mechanism | Lease/heartbeat + Witness reclamation |
| Observability | CLI only | HTTP `/health`, `/metrics`, `/status` |