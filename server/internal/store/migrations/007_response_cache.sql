CREATE TABLE IF NOT EXISTS kv_cache (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);