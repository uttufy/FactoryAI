Based only on [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md), the biggest architectural improvement is to make the system simpler, more layered, and more explicit about ownership.

**Main Architectural Improvements**

1. Split the system into 4 clear layers.
- `Data plane`: Beads, workflow state, worker state, merge queue state.
- `Control plane`: scheduling, dispatch, retries, patrol logic.
- `Execution plane`: worker runtime, Claude sessions, handoff, nudging.
- `Interface plane`: CLI, TUI, automation commands.

Right now `PlantManager`, `Deacon`, `Witness`, `Refinery`, `HandoffManager`, and `NudgeService` overlap too much in responsibility ([plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):602, [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):646, [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):666, [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):770, [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):796).

2. Make one component the owner of work dispatch.
- Add a dedicated `Scheduler/Dispatcher` service.
- Only that service should claim work, attach hooks, assign workers, and advance workflow execution.
- `Witness` and `Deacon` should observe and request actions, not directly compete to mutate the same work state.

This removes race conditions between “check hook”, “nudge”, “handoff”, and “dispatch” behaviors.

3. Make Beads the single source of truth for durable state.
- `plan.md` says the system is durable and survives crashes/restarts ([plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):879), but the architecture does not define which state is authoritative.
- Beads should store durable work state.
- tmux should not be a source of truth.
- In-memory maps should only be caches.

4. Replace tmux-first execution with a runtime abstraction.
- Introduce something like `AgentRuntime`.
- tmux becomes one adapter implementation, not the core architecture.
- Handoff, capture, nudge, and resume should target the runtime abstraction, not tmux directly.

That makes the architecture more portable and much less fragile than tying orchestration to pane/session management ([plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):571).

5. Redesign workflows around runs, not just templates.
- Keep `Formula` and `Protomolecule` as templates.
- Add explicit runtime entities like `WorkflowRun` and `StepRun`.
- Store claim status, retries, timestamps, outputs, and recovery metadata there.

Right now `Molecule` mixes template-like and runtime-like concerns, and `Dependencies` conflicts with `CurrentStep` ([plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):385, [plan.md](/Users/utkarshsharma/projects/FactoryAI/plan.md):416).

6. Choose one workflow engine model for v1.
- Either linear workflows with `CurrentStep`
- Or DAG workflows with dependency scheduling

Keeping both in the same first architecture will make retries, crash recovery, and step claiming much harder.

7. Add explicit state machines for core entities.
The architecture should define transitions for:
- `Bead`
- `WorkOrder`
- `Worker`
- `Molecule/WorkflowRun`
- `MergeRequest`
- `Convoy`

Right now the interfaces exist, but the transition rules do not. That is risky for a distributed multi-agent system.

8. Separate commands from events.
- Commands: attach work, claim step, retry, merge, escalate
- Events: work attached, step started, step completed, step failed, merge blocked

This will make patrol loops, mail, convoy tracking, and observability much cleaner than direct service-to-service mutation.

9. Make merge queue a standalone subsystem.
- Workers should produce artifacts: branch, patch, result summary, verification metadata.
- `Refinery` should consume those artifacts and own merge state.
- `Convoy` should track business-level progress, not act like another execution engine.

10. Reduce the architecture for v1.
The current plan includes too many top-level concepts at once:
- Beads
- hooks
- mail
- convoys
- molecules
- formulas
- swarms
- refinery
- witness
- deacon
- dogs
- nudge
- handoff

A stronger architecture would build in this order:
1. Beads-backed durable work state
2. Worker runtime
3. Hooks/dispatch
4. Workflow engine
5. Merge queue
6. Patrol agents
7. Convoys/mail/swarms

**Best Single Improvement**
If I had to pick one: add a central `Scheduler/Dispatcher` and make every other role either:
- state owner, or
- observer/operator

That one change would make the whole architecture much cleaner.

If you want, I can turn this into a revised target architecture diagram for `plan.md` with exact components and responsibilities.