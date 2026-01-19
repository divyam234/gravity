package store

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gravity-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s, err := New(tempDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Verify table exists by attempting a query
	var name string
	err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='downloads'").Scan(&name)
	if err != nil {
		t.Errorf("failed to find downloads table: %v", err)
	}
	if name != "downloads" {
		t.Errorf("expected table downloads, got %s", name)
	}
}
