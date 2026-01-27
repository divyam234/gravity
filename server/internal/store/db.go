package store

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gravity/internal/config"
	"gravity/internal/logger"
	"gravity/internal/model"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

// StatsKV represents a simple key-value store for statistics
type StatsKV struct {
	Key       string `gorm:"primaryKey"`
	Value     int64
	UpdatedAt time.Time
}

func (StatsKV) TableName() string { return "stats" }

func New(cfg *config.Config) (*Store, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Type {
	case "postgres":
		dialector = postgres.Open(cfg.Database.DSN)
	case "sqlite", "":
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		// Enable WAL mode and set busy timeout for better concurrency
		dsn := cfg.Database.DSN
		if dsn != ":memory:" {
			if !strings.Contains(dsn, "?") {
				dsn += "?_journal_mode=WAL&_busy_timeout=5000"
			} else {
				if !strings.Contains(dsn, "_journal_mode") {
					dsn += "&_journal_mode=WAL"
				}
				if !strings.Contains(dsn, "_busy_timeout") {
					dsn += "&_busy_timeout=5000"
				}
			}
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	newLogger := gormlogger.New(
		logger.ZapWriter{Sugar: logger.S}, // Use ZapWriter wrapper
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.L.Debug("DB: Connection established. Running AutoMigrate...")
	// RUN AUTOMIGRATE
	if err := db.AutoMigrate(
		&model.Download{},
		&model.DownloadFile{},
		&model.Provider{},
		&model.Settings{},
		&StatsKV{},
		&model.IndexedFile{},
		&model.RemoteIndexConfig{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	logger.L.Debug("DB: AutoMigrate complete. Configuring connection pool...")
	sqlDB, err := db.DB()
	if err == nil {
		if cfg.Database.Type == "sqlite" || cfg.Database.Type == "" {
			sqlDB.SetMaxOpenConns(1)
		} else {
			sqlDB.SetMaxOpenConns(25)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(time.Hour)
		}
	}

	s := &Store{db: db}

	// Legacy SQL migrations: ONLY for SQLite FTS which AutoMigrate can't handle
	if cfg.Database.Type == "sqlite" || cfg.Database.Type == "" {
		logger.L.Debug("DB: Setting up SQLite FTS index...")
		if err := s.setupSQLiteFTS(); err != nil {
			logger.L.Warn("SQLite FTS setup failed", zap.Error(err))
		}
	}

	// PostgreSQL-specific FTS index
	if cfg.Database.Type == "postgres" {
		logger.L.Debug("DB: Creating Postgres FTS index...")
		if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_indexed_files_fts ON indexed_files USING GIN (to_tsvector('english', filename || ' ' || path));`).Error; err != nil {
			logger.L.Warn("Postgres FTS index creation failed", zap.Error(err))
		}
	}

	logger.L.Info("DB: Initialization complete.")
	return s, nil
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) GetDB() *gorm.DB {
	return s.db
}

func (s *Store) setupSQLiteFTS() error {
	// Custom SQL for FTS5 (AutoMigrate doesn't support virtual tables)
	stmts := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS files_search USING fts5(filename, path, remote, content='indexed_files', content_rowid='rowid');`,
		`DROP TRIGGER IF EXISTS indexed_files_ai;`,
		`CREATE TRIGGER indexed_files_ai AFTER INSERT ON indexed_files BEGIN
			INSERT INTO files_search(rowid, filename, path, remote) VALUES (new.rowid, new.filename, new.path, new.remote);
		END;`,
		`DROP TRIGGER IF EXISTS indexed_files_ad;`,
		`CREATE TRIGGER indexed_files_ad AFTER DELETE ON indexed_files BEGIN
			INSERT INTO files_search(files_search, rowid, filename, path, remote) VALUES('delete', old.rowid, old.filename, old.path, old.remote);
		END;`,
		`DROP TRIGGER IF EXISTS indexed_files_au;`,
		`CREATE TRIGGER indexed_files_au AFTER UPDATE ON indexed_files BEGIN
			INSERT INTO files_search(files_search, rowid, filename, path, remote) VALUES('delete', old.rowid, old.filename, old.path, old.remote);
			INSERT INTO files_search(rowid, filename, path, remote) VALUES (new.rowid, new.filename, new.path, new.remote);
		END;`,
	}
	for _, stmt := range stmts {
		if err := s.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
