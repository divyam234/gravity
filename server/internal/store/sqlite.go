package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "gravity.db")
	// Add SQLite connection parameters for better concurrency and reliability
	// _busy_timeout=5000: Wait up to 5s if DB is locked instead of failing immediately
	// _journal_mode=WAL: Enable Write-Ahead Logging for concurrent read/write
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set performance and safety PRAGMAs
	if _, err := db.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous pragma: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to set foreign_keys pragma: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Limit to 1 connection to prevent "database is locked" errors.
	// SQLite handles concurrent reads/writes better when Go serializes the writes.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetDB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		content, err := migrationsFS.ReadFile("migrations/" + f)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		// Robust statement splitting that handles BEGIN...END blocks (like triggers)
		var statements []string
		var current strings.Builder
		inBegin := false

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}

			current.WriteString(line)
			current.WriteString("\n")

			upper := strings.ToUpper(trimmed)
			if strings.Contains(upper, "BEGIN") {
				inBegin = true
			}
			if strings.Contains(upper, "END") {
				inBegin = false
			}

			if !inBegin && strings.HasSuffix(trimmed, ";") {
				statements = append(statements, current.String())
				current.Reset()
			}
		}

		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := s.db.Exec(stmt); err != nil {
				// Ignore "duplicate column name" or "table already exists" errors
				if strings.Contains(err.Error(), "duplicate column name") ||
					strings.Contains(err.Error(), "already exists") {
					continue
				}
				return fmt.Errorf("failed to execute migration %s: %w (statement: %s)", f, err, stmt)
			}
		}
	}

	return nil
}
