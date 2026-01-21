-- Source of truth for cloud file metadata
CREATE TABLE indexed_files (
    id TEXT PRIMARY KEY,
    remote TEXT NOT NULL,
    path TEXT NOT NULL,
    filename TEXT NOT NULL,
    size INTEGER,
    mod_time DATETIME,
    is_dir BOOLEAN,
    last_indexed_at DATETIME
);

-- Virtual table for Full-Text Search
CREATE VIRTUAL TABLE files_search USING fts5(
    filename,
    path,
    remote,
    content='indexed_files',
    content_rowid='rowid'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER indexed_files_ai AFTER INSERT ON indexed_files BEGIN
  INSERT INTO files_search(rowid, filename, path, remote) VALUES (new.rowid, new.filename, new.path, new.remote);
END;

CREATE TRIGGER indexed_files_ad AFTER DELETE ON indexed_files BEGIN
  INSERT INTO files_search(files_search, rowid, filename, path, remote) VALUES('delete', old.rowid, old.filename, old.path, old.remote);
END;

CREATE TRIGGER indexed_files_au AFTER UPDATE ON indexed_files BEGIN
  INSERT INTO files_search(files_search, rowid, filename, path, remote) VALUES('delete', old.rowid, old.filename, old.path, old.remote);
  INSERT INTO files_search(rowid, filename, path, remote) VALUES (new.rowid, new.filename, new.path, new.remote);
END;

-- Populate index from existing data
INSERT INTO files_search(rowid, filename, path, remote)
SELECT rowid, filename, path, remote FROM indexed_files;

-- Indexing configuration per remote
CREATE TABLE remote_index_config (
    remote TEXT PRIMARY KEY,
    auto_index_interval_mins INTEGER DEFAULT 0, -- 0 = disabled
    last_indexed_at DATETIME,
    status TEXT DEFAULT 'idle', -- idle, indexing, error
    error_msg TEXT
);
