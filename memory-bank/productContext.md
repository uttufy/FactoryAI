# Product Context: FactoryAI

## Problem Statement
Software development often involves multiple parallel tasks that could benefit from AI assistance. However, managing multiple AI agents working simultaneously on different parts of a codebase is complex and requires:
- Isolation between work items to prevent conflicts
- Coordination of work across multiple agents
- Tracking progress and handling failures
- Merging completed work back into the main branch

## Solution
FactoryAI applies manufacturing factory concepts to software development:
- **Stations** provide isolated git worktrees for parallel development
- **Operators** (AI agents) work independently at each station
- **Travelers** track work progress through the factory
- **SOPs** (Standard Operating Procedures) define repeatable workflows
- **Event Bus** enables reactive communication between components

## Target Users
- Development teams wanting to parallelize AI-assisted coding
- Individual developers managing multiple concurrent tasks
- Teams with complex multi-step development workflows

## Key Features
1. **Parallel Execution**: Multiple stations working simultaneously
2. **Work Isolation**: Git worktrees prevent conflicts during development
3. **DAG Workflows**: Define complex multi-step procedures with dependencies
4. **Crash Recovery**: SQLite-based lease system for fault tolerance
5. **Quality Control**: Inspection and rework loops
6. **Merge Management**: Final assembly with conflict detection

## User Experience Goals
- Simple CLI interface for factory management
- Real-time TUI dashboard for monitoring
- Minimal configuration required to get started
- Clear visibility into factory status and work progress