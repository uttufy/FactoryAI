-- 001_init.sql
-- Initial schema for FactoryAI v1.0 Production Log

-- Stations table
CREATE TABLE IF NOT EXISTS stations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    worktree_path TEXT,
    tmux_session TEXT,
    tmux_window INTEGER,
    tmux_pane INTEGER,
    current_job TEXT,
    operator_id TEXT,
    created_at DATETIME NOT NULL,
    last_activity DATETIME NOT NULL
);

-- Operators table
CREATE TABLE IF NOT EXISTS operators (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    station_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_task TEXT,
    claude_session TEXT,
    started_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL,
    completed_at DATETIME,
    skills TEXT,
    FOREIGN KEY (station_id) REFERENCES stations(id)
);

-- Leases table for crash recovery
CREATE TABLE IF NOT EXISTS leases (
    id TEXT PRIMARY KEY,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    acquired_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_leases_expires ON leases(expires_at);
CREATE INDEX IF NOT EXISTS idx_leases_resource ON leases(resource_type, resource_id);

-- SOPs (Standard Operating Procedures) table
CREATE TABLE IF NOT EXISTS sops (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    steps TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    completed_at DATETIME,
    is_wisp INTEGER DEFAULT 0
);

-- Travelers table
CREATE TABLE IF NOT EXISTS travelers (
    id TEXT PRIMARY KEY,
    station_id TEXT NOT NULL UNIQUE,
    bead_id TEXT NOT NULL,
    sop_id TEXT,
    priority INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    deferred INTEGER DEFAULT 0,
    restart INTEGER DEFAULT 0,
    rework_count INTEGER DEFAULT 0,
    rework_reason TEXT,
    attached_at DATETIME NOT NULL,
    started_at DATETIME,
    completed_at DATETIME,
    result TEXT,
    error TEXT,
    FOREIGN KEY (station_id) REFERENCES stations(id)
);

-- Event log for replay/debugging
CREATE TABLE IF NOT EXISTS event_log (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    source TEXT,
    subject TEXT,
    payload TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp ON event_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON event_log(type);

-- Dead letter queue for dropped events
CREATE TABLE IF NOT EXISTS dead_letter (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    reason TEXT
);

-- Batches table
CREATE TABLE IF NOT EXISTS batches (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    tracked_ids TEXT,
    work_cells TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    completed_at DATETIME,
    result TEXT
);

-- Merge requests table
CREATE TABLE IF NOT EXISTS merge_requests (
    id TEXT PRIMARY KEY,
    bead_id TEXT NOT NULL,
    station_id TEXT NOT NULL,
    branch TEXT NOT NULL,
    status TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    conflicts TEXT,
    submitted_at DATETIME NOT NULL,
    merged_at DATETIME,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_mr_status ON merge_requests(status);

-- Factory status table for process-isolated state management
-- This table enables cross-process state tracking since each CLI command
-- runs as a separate process with isolated memory
CREATE TABLE IF NOT EXISTS factory_status (
    id INTEGER PRIMARY KEY CHECK (id = 1),  -- Singleton row
    running INTEGER NOT NULL DEFAULT 0,
    pid INTEGER,
    started_at DATETIME,
    boot_status TEXT NOT NULL DEFAULT 'stopped'  -- 'booting', 'running', 'shutting_down', 'stopped'
);

-- Initialize factory status row
INSERT OR IGNORE INTO factory_status (id, running, boot_status) VALUES (1, 0, 'stopped');
