-- Add filter columns to search indexing configuration
ALTER TABLE remote_index_config ADD COLUMN excluded_patterns TEXT; -- Regex or glob patterns
ALTER TABLE remote_index_config ADD COLUMN included_extensions TEXT; -- Comma separated extensions
ALTER TABLE remote_index_config ADD COLUMN min_size_bytes INTEGER DEFAULT 0;
