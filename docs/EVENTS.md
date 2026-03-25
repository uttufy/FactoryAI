# FactoryAI Events System (Andon Board)

## Overview

The Andon Board is FactoryAI's event-driven communication system. It implements a pub/sub pattern for real-time communication between components, inspired by manufacturing andon cords that signal issues on the factory floor.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     ANDON BOARD (Event Bus)                 │
│  ────────────────────────────────────────────────────────  │
│  - In-memory pub/sub                                       │
│  - Per-event-type FIFO ordering                            │
│  - Non-blocking publish                                    │
│  - Dead letter queue for dropped events                    │
│  - Event logging to SQLite                                 │
└────────┬────────────────────────────────────────────────────┘
         │
    ┌────┴────┬────────┬────────┬────────┬────────┐
    │         │        │        │        │        │
    ▼         ▼        ▼        ▼        ▼        ▼
 Director  Planner  Engineer Supervisor Support Assembly
```

## Event Types

### Job Lifecycle Events

#### `JobCreated`

Emitted when a new job is created.

```go
Event{
    Type: JobCreated,
    Source: "director",
    Target: beadID,
    Payload: {
        "task": task,
        "priority": priority,
        "formula": formulaPath,
    },
}
```

**Subscribers:** None (informational)

#### `JobQueued`

Emitted when a job is added to the work queue.

```go
Event{
    Type: JobQueued,
    Source: "planner",
    Target: beadID,
    Payload: {
        "priority": priority,
        "queued_at": timestamp,
    },
}
```

**Subscribers:** Planner

### Step Execution Events

#### `StepStarted`

Emitted when a processing step begins.

```go
Event{
    Type: StepStarted,
    Source: "planner",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "step_name": stepName,
        "operator_id": operatorID,
    },
}
```

**Subscribers:** Director, TUI

#### `StepCompleted`

Emitted when a processing step completes successfully.

```go
Event{
    Type: StepCompleted,
    Source: "station",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "step_name": stepName,
        "output": output,
        "duration": duration,
    },
}
```

**Subscribers:** Planner, Inspector, TUI

#### `StepFailed`

Emitted when a processing step fails.

```go
Event{
    Type: StepFailed,
    Source: "station",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "step_name": stepName,
        "error": error,
        "retry_count": retryCount,
        "max_retries": maxRetries,
    },
}
```

**Subscribers:** Planner, Supervisor, TUI

### Station Events

#### `StationReady`

Emitted when a station becomes ready for work.

```go
Event{
    Type: StationReady,
    Source: "station_manager",
    Target: stationID,
    Payload: {
        "status": "idle",
        "available": true,
    },
}
```

**Subscribers:** Planner

#### `StationOffline`

Emitted when a station goes offline.

```go
Event{
    Type: StationOffline,
    Source: "supervisor",
    Target: stationID,
    Payload: {
        "reason": reason,
        "last_seen": timestamp,
    },
}
```

**Subscribers:** Supervisor, Support

### Operator Events

#### `OperatorSpawned`

Emitted when a new operator is spawned.

```go
Event{
    Type: OperatorSpawned,
    Source: "operator_pool",
    Target: operatorID,
    Payload: {
        "station_id": stationID,
        "role": role,
        "model": model,
    },
}
```

**Subscribers:** Director, TUI

#### `OperatorStuck`

Emitted when an operator is detected as stuck.

```go
Event{
    Type: OperatorStuck,
    Source: "supervisor",
    Target: operatorID,
    Payload: {
        "station_id": stationID,
        "last_heartbeat": timestamp,
        "stuck_duration": duration,
        "nudge_message": message,
    },
}
```

**Subscribers:** Supervisor, Support

#### `OperatorHandoff`

Emitted during operator handoff between stations.

```go
Event{
    Type: OperatorHandoff,
    Source: "supervisor",
    Target: operatorID,
    Payload: {
        "from_station": fromStationID,
        "to_station": toStationID,
        "handoff_time": timestamp,
    },
}
```

**Subscribers:** Supervisor

### Merge Events

#### `MergeReady`

Emitted when work is ready for merging.

```go
Event{
    Type: MergeReady,
    Source: "traveler",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "branch": branchName,
        "ready_at": timestamp,
    },
}
```

**Subscribers:** Assembly

#### `MergeStarted`

Emitted when a merge begins.

```go
Event{
    Type: MergeStarted,
    Source: "assembly",
    Target: mergeID,
    Payload: {
        "branch": branchName,
        "conflict_check": result,
    },
}
```

**Subscribers:** TUI

#### `MergeCompleted`

Emitted when a merge completes successfully.

```go
Event{
    Type: MergeCompleted,
    Source: "assembly",
    Target: mergeID,
    Payload: {
        "branch": branchName,
        "merged_at": timestamp,
        "commit": commitHash,
    },
}
```

**Subscribers:** Director, TUI

#### `MergeConflict`

Emitted when a merge has conflicts.

```go
Event{
    Type: MergeConflict,
    Source: "assembly",
    Target: mergeID,
    Payload: {
        "branch": branchName,
        "conflicts": []string{"file1.go", "file2.go"},
        "error": error,
    },
}
```

**Subscribers:** Director, Supervisor

### Quality Events

#### `QualityFailed`

Emitted when quality inspection fails.

```go
Event{
    Type: QualityFailed,
    Source: "inspector",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "criteria": criteria,
        "output": output,
        "reason": reason,
    },
}
```

**Subscribers:** Director, Supervisor

#### `ReworkNeeded`

Emitted when rework is required.

```go
Event{
    Type: ReworkNeeded,
    Source: "inspector",
    Target: beadID,
    Payload: {
        "station_id": stationID,
        "reason": reason,
        "priority": 100,
    },
}
```

**Subscribers:** Planner

### System Events

#### `HealthOK`

Emitted when health check passes.

```go
Event{
    Type: HealthOK,
    Source: "support_service",
    Target: "factory",
    Payload: {
        "timestamp": timestamp,
        "checks": checks,
    },
}
```

**Subscribers:** Director

#### `CleanupDone`

Emitted when cleanup operation completes.

```go
Event{
    Type: CleanupDone,
    Source: "support_service",
    Target: taskID,
    Payload: {
        "expired_leases": count,
        "old_events": count,
        "cleaned_at": timestamp,
    },
}
```

**Subscribers:** Director

## Subscription Matrix

| Event | Director | Planner | Engineer | Supervisor | Support | Assembly | TUI |
|-------|----------|---------|----------|------------|---------|----------|-----|
| JobCreated | ✓ | - | - | - | - | - | - |
| JobQueued | - | Subscribe | - | - | - | - | - |
| StepStarted | ✓ | - | - | - | - | - | ✓ |
| StepCompleted | ✓ | Subscribe | - | - | - | - | ✓ |
| StepFailed | ✓ | ✓ | - | Subscribe | - | - | ✓ |
| StationReady | - | Subscribe | - | - | - | - | - |
| StationOffline | - | - | - | Subscribe | Subscribe | - | ✓ |
| OperatorSpawned | ✓ | - | - | - | - | - | ✓ |
| OperatorStuck | - | - | - | Subscribe | Subscribe | - | ✓ |
| OperatorHandoff | - | - | - | Subscribe | - | - | - |
| MergeReady | - | - | - | - | - | Subscribe | - |
| MergeStarted | ✓ | - | - | - | - | - | ✓ |
| MergeCompleted | ✓ | - | - | - | - | - | ✓ |
| MergeConflict | ✓ | - | - | Subscribe | - | - | ✓ |
| QualityFailed | ✓ | - | - | Subscribe | - | - | ✓ |
| ReworkNeeded | - | Subscribe | - | - | - | - | - |
| HealthOK | Subscribe | - | - | - | - | - | - |
| CleanupDone | Subscribe | - | - | - | - | - | - |

## Event Properties

### Durability

Events are **in-memory only** for v1.0. They are logged to SQLite for recovery but not persisted before delivery.

### Ordering

**Per-event-type FIFO**: Within the same event type, order is preserved. Different event types may be delivered out of order.

### Delivery

**At-most-once**: Events are delivered once. If a handler crashes, the event is not redelivered.

### Backpressure

**Drop + log**: If the buffer is full, events are dropped and logged to the dead letter queue.

### Blocking

**Non-blocking publish**: Publishers never block. If the buffer is full, the event is dropped.

## Dead Letter Queue

Events that cannot be delivered are logged to the dead letter queue:

```go
DeadLetterEvent{
    OriginalEvent: event,
    DroppedAt:     timestamp,
    Reason:        "buffer_full",
}
```

The dead letter queue can be queried:

```bash
factory support status --dead-letter
```

## Event Logging

All events are logged to SQLite for recovery and debugging:

```sql
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    source TEXT NOT NULL,
    target TEXT NOT NULL,
    payload TEXT,  -- JSON
    created_at INTEGER NOT NULL,
    processed BOOLEAN DEFAULT FALSE
);
```

Query event log:

```bash
factory events list --type StepCompleted --limit 100
```

## Using Events in Code

### Emitting Events

```go
import "github.com/uttufy/FactoryAI/internal/events"

// In your component
func (s *MyService) DoWork(ctx context.Context) error {
    // Emit event before work
    s.events.Emit(events.EventStepStarted, "my_service", beadID, map[string]interface{}{
        "station_id": stationID,
    })

    // Do work
    result, err := s.doActualWork(ctx)
    if err != nil {
        s.events.Emit(events.EventStepFailed, "my_service", beadID, map[string]interface{}{
            "error": err.Error(),
        })
        return err
    }

    // Emit completion
    s.events.Emit(events.EventStepCompleted, "my_service", beadID, map[string]interface{}{
        "output": result,
    })
    return nil
}
```

### Subscribing to Events

```go
import "github.com/uttufy/FactoryAI/internal/events"

func (p *Planner) Start(ctx context.Context) error {
    // Subscribe to specific event type
    p.events.Subscribe(events.EventJobQueued, p.handleJobQueued)

    // Or subscribe to all events
    p.events.SubscribeAll(p.handleAnyEvent)

    return nil
}

func (p *Planner) handleJobQueued(evt events.Event) {
    beadID, ok := evt.Payload["bead_id"].(string)
    if !ok {
        return
    }

    // Handle the event
    p.processJob(beadID)
}
```

### Unsubscribing

```go
func (p *Planner) Stop(ctx context.Context) error {
    // Unsubscribe from specific handler
    p.events.Unsubscribe(events.EventJobQueued, p.handleJobQueued)
    return nil
}
```

## Event Flow Examples

### Job Completion Flow

```
1. User: factory job create "task"
2. Director: Emit(JobCreated)
3. Director: Create bead
4. Director: Create batch
5. Planner: Enqueue bead
6. Planner: Emit(JobQueued)
7. Planner: ProcessQueue()
8. Planner: Find available station
9. Planner: Attach traveler
10. Planner: Emit(StepStarted)
11. Operator: Execute work
12. Operator: Complete work
13. Station: Emit(StepCompleted)
14. Inspector: Inspect work
15. If PASS:
16.   Inspector: Emit(QualityPassed)
17.   Traveler: Complete
18.   Assembly: Submit for merge
19.   Assembly: Emit(MergeReady)
20. If FAIL:
21.   Inspector: Emit(QualityFailed)
22.   Inspector: Emit(ReworkNeeded)
23.   Planner: Requeue for rework
```

### Operator Stuck Flow

```
1. Support: Periodic health check
2. Support: Check operator heartbeats
3. Support: Find stuck operator
4. Support: Emit(OperatorStuck)
5. Supervisor: Receive OperatorStuck event
6. Supervisor: Attempt recovery
7. Supervisor: Send nudge
8. If recovered:
9.   Operator: Resume work
10. If not recovered:
11.  Supervisor: Emit(OperatorStuck) again
12.  Director: Escalate to human
```

## Best Practices

### 1. Event Naming

Use past tense for completed events, present tense for ongoing:

- ✅ `StepCompleted`, `StepFailed`, `JobCreated`
- ✅ `OperatorStuck`, `MergeConflict`
- ❌ `StepComplete`, `JobCreate`

### 2. Payload Structure

Include relevant context in payload:

```go
// Good: Detailed context
map[string]interface{}{
    "station_id": stationID,
    "step_name": stepName,
    "duration": duration.Milliseconds(),
    "output_size": len(output),
}

// Bad: Minimal context
map[string]interface{}{
    "done": true,
}
```

### 3. Event Size

Keep payloads reasonable:

```go
// Good: Reference to data
map[string]interface{}{
    "output_path": "/tmp/output.txt",
    "line_count": 1000,
}

// Bad: Entire output in event
map[string]interface{}{
    "output": hugeString,  // Don't do this
}
```

### 4. Error Handling

Never panic in event handlers:

```go
func (p *Planner) handleJobQueued(evt events.Event) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered in handler: %v", r)
        }
    }()

    // Handle event
}
```

### 5. Unsubscribing

Always clean up subscriptions:

```go
func (s *MyService) Stop(ctx context.Context) error {
    s.events.UnsubscribeAll(s)
    return nil
}
```

## Debugging Events

### Enable Event Logging

```bash
# Enable debug logging
export FACTORY_LOG_LEVEL=debug

# Run with verbose output
factory status --verbose
```

### Query Event Log

```bash
# List recent events
factory events list --limit 50

# Filter by type
factory events list --type StepCompleted

# Filter by bead
factory events list --bead bead-123

# Filter by time range
factory events list --since "1h ago"
```

### Monitor Dead Letter

```bash
# Check dead letter queue
factory support status --dead-letter

# View dropped events
factory events list --dead-letter
```

## Performance Considerations

### Buffer Size

Default buffer size is 1000 events. Adjust in config:

```yaml
events:
  buffer_size: 5000  # Increase for high throughput
```

### Handler Duration

Keep handlers fast. Long-running work should be async:

```go
func (p *Planner) handleJobQueued(evt events.Event) {
    // Fast: just queue the work
    p.workQueue <- evt

    // NOT this: blocks the event bus
    // time.Sleep(10 * time.Second)
}
```

### Subscription Count

Minimize subscriptions. Each event is delivered to all subscribers:

```go
// Good: One subscriber handles multiple concerns
events.SubscribeAll(func(evt events.Event) {
    // Handle based on type
    switch evt.Type {
    case EventA:
        handleA(evt)
    case EventB:
        handleB(evt)
    }
})

// Bad: Multiple subscriptions for same component
events.Subscribe(EventA, handleA)
events.Subscribe(EventB, handleB)
events.Subscribe(EventC, handleC)
```

## See Also

- [Architecture Documentation](ARCHITECTURE.md)
- [CLI Reference](CLI.md)
- [Formulas Guide](FORMULAS.md)
