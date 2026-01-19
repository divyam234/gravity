-- 002_magnet_files.sql

-- Add magnet fields to downloads table (SQLite requires separate ALTER statements)
-- Using IF NOT EXISTS pattern via checking sqlite_master to be idempotent

-- Create download_files table for individual file tracking
CREATE TABLE IF NOT EXISTS download_files (
    id TEXT PRIMARY KEY,
    download_id TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    downloaded INTEGER NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    engine_id TEXT,
    url TEXT,
    file_index INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (download_id) REFERENCES downloads(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_download_files_download_id ON download_files(download_id);
CREATE INDEX IF NOT EXISTS idx_download_files_engine_id ON download_files(engine_id);
