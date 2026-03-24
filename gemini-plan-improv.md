Focusing specifically on the high-level Architecture defined in your diagram and LLD, here are the core structural improvements to make the system more scalable, resilient, and event-driven:

1. Introduce an Event Bus (Remove Polling / Patrol Loops)
Current state: The architecture heavily relies on active "Patrol Loops" (e.g., Deacon and Floor Monitor/Witness constantly polling to check if workers are stuck or if work is ready).
The flaw: Polling is resource-intensive, introduces latency, and scales poorly as the number of workers increases.
Architectural Improvement: Introduce an internal Event Bus / PubSub layer. Instead of the Floor Monitor looping to check on workers, a Worker should emit an event (e.g., StepFailed, WaitingForInput, TaskComplete) to the Event Bus. The Witness and Merge Station simply subscribe to these events and react instantly.
2. Add an explicit "Sandbox/Isolation Manager" to Production Lines
Current state: A Production Line (Rig) contains multiple ephemeral Workers and Floor Crew attacking the same Rig (codebase).
The flaw: If three AI agents are running side-by-side in the same local Git directory via Tmux panes, they will cause catastrophic collisions (overwriting files, breaking lockfiles, causing npm test or go build to fail for each other due to dirty state).
Architectural Improvement: The Architecture needs a Workspace Management Layer within the Production Line. Before assigning a Worker, this manager provisions an isolated environment (e.g., a distinct Git worktree, a temporary directory clone, or an isolated Docker container). The Worker operates there, and the Merge Station is responsible for safely pulling those isolated changes back into the main Rig.
3. Decouple the Plant Manager (Resolve the SPOF)
Current state: The Plant Manager (Mayor) acts as a centralized brain—receiving requests, dispatching work, monitoring status, and managing Convoys.
The flaw: It's a Single Point of Failure and a potential bottleneck. If the Mayor process locks up or crashes, all orchestration halts.
Architectural Improvement: Split the Plant Manager into two decoupled components:
Ingestion/API Gateway: Handles incoming user commands, CLI inputs, and eventual webhooks, dumping them into a durable Queue.
Scheduler/Dispatcher: Reads from the Queue and assigns work. Production Lines can actually pull work from this queue when they have idle capacity, rather than having the Mayor actively push/dispatch to them.
4. Provide a Factory State Store (Separate from Beads)
Current state: Work items are stored in Beads (the MEOW stack), but the runtime execution state (e.g., which worker is on which tmux pane, what phase a handoff is in, active Swarm mappings) only lives in Go memory maps.
The flaw: Reboots, crashes, or updates to the FactoryAI binary will wipe out all knowledge of in-progress operations, leaving zombie tmux sessions running Claude instances infinitely.
Architectural Improvement: Add a local Runtime KV Data Store (like SQLite, BoltDB, or a rigid .factory/state.json) attached to the Plant Manager. Every time a worker is spawned into a Tmux pane or a Convoy changes state, it is committed to this store. This allows your Watchdog (Boot) to instantly recover and re-attach to the whole factory state upon restart.
5. Standardize Agent Interfaces (The "Multi-Model" Layer)
Current state: The agents are presumed to be raw claude (Claude Code) CLI processes managed in Tmux.
The flaw: The orchestration is tightly coupled to how Claude Code behaves. This makes adding OpenAI, open-source models, or specialized agents (like a purely AST-based refactoring agent) difficult.
Architectural Improvement: Introduce an Agent Adapter API Boundary. Instead of the Factory managing "Claude Sessions" or sending raw keystrokes to a pane, the Factory communicates via JSON/RPC to a standard Agent API. The Adapter handles translating universal commands (like "Pause", "Resume", "Status") into the specific keystrokes or API calls needed for Claude, ChatGPT-o1, or any future tool.
