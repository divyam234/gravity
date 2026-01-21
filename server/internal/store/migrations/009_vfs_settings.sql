-- 009_vfs_settings.sql

-- Initialize VFS settings
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('vfs_cache_mode', 'off'),
    ('vfs_cache_max_size', '10G'),
    ('vfs_cache_max_age', '1h'),
    ('vfs_write_back', '5s'),
    ('vfs_read_chunk_size', '128M'),
    ('vfs_read_chunk_size_limit', 'off'),
    ('vfs_read_ahead', '128M'),
    ('vfs_dir_cache_time', '5m'),
    ('vfs_poll_interval', '1m'),
    ('vfs_read_chunk_streams', '0');

-- Initialize other missing group settings if any
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('max_download_speed', '0'),
    ('max_upload_speed', '0'),
    ('max_connection_per_server', '16'),
    ('split', '16'),
    ('user_agent', 'gravity/1.0'),
    ('proxy_enabled', 'false'),
    ('proxy_url', ''),
    ('proxy_user', ''),
    ('proxy_password', ''),
    ('seed_ratio', '1.0'),
    ('seed_time', '1440'),
    ('listen_port', '6881'),
    ('force_save', 'false'),
    ('enable_pex', 'true'),
    ('enable_dht', 'true'),
    ('enable_lpd', 'true'),
    ('bt_encryption', 'plain'),
    ('connect_timeout', '60'),
    ('max_tries', '5'),
    ('check_certificate', 'true');
