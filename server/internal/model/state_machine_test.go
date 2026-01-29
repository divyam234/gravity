package model

import (
	"testing"
)

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    DownloadStatus
		to      DownloadStatus
		wantErr bool
	}{
		// Happy paths
		{"Initial to Waiting", "", StatusWaiting, false},
		{"Waiting to Allocating", StatusWaiting, StatusAllocating, false},
		{"Allocating to Active", StatusAllocating, StatusActive, false},
		{"Allocating to Paused", StatusAllocating, StatusPaused, false}, // New
		{"Active to Complete", StatusActive, StatusComplete, false},
		{"Active to Error", StatusActive, StatusError, false},
		{"Error to Waiting (Retry)", StatusError, StatusWaiting, false},
		{"Complete to Waiting (Retry)", StatusComplete, StatusWaiting, false},

		// Invalid paths
		{"Initial to Active", "", StatusActive, true},
		{"Waiting to Complete", StatusWaiting, StatusComplete, true},
		{"Complete to Active", StatusComplete, StatusActive, true},
		{"Error to Active", StatusError, StatusActive, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateTransition(tt.from, tt.to); (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDownload_TransitionTo(t *testing.T) {
	d := &Download{
		Status: StatusWaiting,
	}

	// Valid transition
	if err := d.TransitionTo(StatusAllocating); err != nil {
		t.Errorf("TransitionTo(StatusAllocating) failed: %v", err)
	}
	if d.Status != StatusAllocating {
		t.Errorf("Status = %v, want %v", d.Status, StatusAllocating)
	}
	if d.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not updated")
	}

	// Invalid transition
	if err := d.TransitionTo(StatusComplete); err == nil {
		t.Error("TransitionTo(StatusComplete) expected error, got nil")
	}
	if d.Status != StatusAllocating {
		t.Errorf("Status changed on error: %v", d.Status)
	}
}
