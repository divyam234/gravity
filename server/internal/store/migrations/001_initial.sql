-- 001_initial.sql

-- Downloads table
CREATE TABLE IF NOT EXISTS downloads (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    resolved_url TEXT,
    provider TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    -- File info
    filename TEXT,
    local_path TEXT,
    size INTEGER DEFAULT 0,
    downloaded INTEGER DEFAULT 0,
    speed INTEGER DEFAULT 0,
    eta INTEGER DEFAULT 0,
    destination TEXT,
    upload_status TEXT,
    upload_progress INTEGER DEFAULT 0,
    upload_speed INTEGER DEFAULT 0,
    category TEXT,
    tags TEXT,
    engine_id TEXT,
    upload_job_id TEXT,
    -- Magnet/torrent fields
    is_magnet INTEGER DEFAULT 0,
    magnet_hash TEXT,
    magnet_source TEXT,
    magnet_id TEXT,
    total_files INTEGER DEFAULT 0,
    files_complete INTEGER DEFAULT 0,
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_created ON downloads(created_at DESC);

-- Providers table
CREATE TABLE IF NOT EXISTS providers (
    name TEXT PRIMARY KEY,
    enabled INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 0,
    config TEXT,
    cached_hosts TEXT,
    cached_account TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Remotes table
CREATE TABLE IF NOT EXISTS remotes (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    config TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Stats table
CREATE TABLE IF NOT EXISTS stats (
    key TEXT PRIMARY KEY,
    value INTEGER DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Initialize default stats
INSERT OR IGNORE INTO stats (key, value) VALUES
    ('total_downloaded', 0),
    ('total_uploaded', 0),
    ('downloads_completed', 0),
    ('uploads_completed', 0),
    ('downloads_failed', 0);

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Initialize default settings
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('download_dir', ''),
    ('max_concurrent_downloads', '5'),
    ('max_concurrent_uploads', '2'),
    ('default_destination', ''),
    ('auto_upload', 'true'),
    ('delete_after_upload', 'false');
