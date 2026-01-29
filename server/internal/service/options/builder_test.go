package options

import (
	"gravity/internal/model"
	"testing"
)

func TestBuilder_Flow(t *testing.T) {
	// 1. Initial Defaults
	builder := NewBuilder()
	opts := builder.Build()
	if *opts.Split != 8 {
		t.Errorf("Expected default split 8, got %d", *opts.Split)
	}

	// 2. Apply Settings (Priority 2)
	settings := &model.Settings{
		Download: model.DownloadSettings{
			Split:       16,
			DownloadDir: "/global/dir",
			UserAgent:   "GlobalAgent",
		},
	}
	builder.WithSettings(settings)
	opts = builder.Build()

	if *opts.Split != 16 {
		t.Errorf("Expected settings split 16, got %d", *opts.Split)
	}
	if opts.DownloadDir != "/global/dir" {
		t.Errorf("Expected settings dir, got %s", opts.DownloadDir)
	}

	// 3. Apply Model Overrides (Priority 3 - Highest)
	dl := &model.Download{
		Dir:   "/task/dir",
		Split: intPtr(4),
	}
	builder.WithModel(dl)
	opts = builder.Build()

	if *opts.Split != 4 {
		t.Errorf("Expected model split 4 (override), got %d", *opts.Split)
	}
	if opts.DownloadDir != "/task/dir" {
		t.Errorf("Expected model dir (override), got %s", opts.DownloadDir)
	}
	// Check that field NOT in model but in settings remains from settings
	if *opts.UserAgent != "GlobalAgent" {
		t.Errorf("Expected UserAgent to persist from settings, got %s", *opts.UserAgent)
	}
}

func TestBuilder_PointerIsolation(t *testing.T) {
	settings := &model.Settings{
		Download: model.DownloadSettings{
			Split: 16,
		},
	}

	builder := NewBuilder().WithSettings(settings)
	opts := builder.Build()

	// Modify the built option (dereferencing and changing)
	*opts.Split = 32

	// Ensure original settings was NOT modified
	if settings.Download.Split != 16 {
		t.Errorf("Original settings was modified! Pointer isolation failed. Expected 16, got %d", settings.Download.Split)
	}
}

func intPtr(v int) *int {
	return &v
}
