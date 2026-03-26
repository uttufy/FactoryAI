# Project Brief: FactoryAI

## Overview
FactoryAI is a multi-agent workspace manager that orchestrates parallel AI agents working on software development tasks. It uses real manufacturing factory concepts and terminology, adapted for software production.

## Core Concept
Transform software development into a manufacturing-style operation where:
- **Stations** = Isolated git worktrees where AI agents work
- **Operators** = AI agents (Claude) working at stations
- **Travelers** = Work orders that move through stations
- **SOPs** = Standard Operating Procedures (DAG workflows)
- **Beads** = Work items managed by the beads CLI

## Key Integrations
- **Beads CLI** (github.com/steveyegge/beads) - Work item management
- **tmux** - Primary UI and session management
- **Claude Code** - The underlying AI agent (`claude` binary)
- **SQLite** - Runtime state storage for crash recovery

## Goals
1. Orchestrate multiple AI agents working in parallel
2. Provide isolated workspaces via git worktrees
3. Enable workflow automation through DAG-based SOPs
4. Support crash recovery and state management
5. Offer real-time monitoring via TUI dashboard

## MVP v1 Architecture (Collapsed Roles)
1. **Plant Director** - User interface, planning, dispatching, DAG evaluation
2. **Floor Supervisor** - Monitoring, inspection, handoffs
3. **Support Service** - Heartbeats, stuck detection, cleanup, nudges

## Technology Stack
- **Language**: Go 1.24.2
- **CLI Framework**: spf13/cobra
- **TUI Framework**: charmbracelet/bubbletea
- **Database**: SQLite (mattn/go-sqlite3)
- **Config Format**: TOML (pelletier/go-toml)
- **UUID**: google/uuid

## Status
- Core business logic: ~95% complete
- CLI entry point: Not implemented
- TUI dashboard: Placeholder only
- Configuration: Not implemented