ALTER TABLE downloads ADD COLUMN priority INTEGER DEFAULT 5;
CREATE INDEX idx_downloads_priority ON downloads(priority);
