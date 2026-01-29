package model

import "fmt"

// ValidTransitions defines allowed state transitions
var ValidTransitions = map[DownloadStatus][]DownloadStatus{
	StatusWaiting:    {StatusAllocating, StatusActive, StatusPaused, StatusError, StatusProcessing},
	StatusAllocating: {StatusActive, StatusError, StatusWaiting, StatusPaused},
	StatusActive:     {StatusPaused, StatusComplete, StatusUploading, StatusError, StatusResolving, StatusWaiting},
	StatusResolving:  {StatusActive, StatusError, StatusWaiting, StatusPaused},
	StatusPaused:     {StatusWaiting, StatusError, StatusActive}, // Added StatusActive for Resume from Allocating/Resolving if needed? No, usually goes to Waiting/Active via Resume
	StatusUploading:  {StatusComplete, StatusError, StatusWaiting},
	StatusComplete:   {StatusWaiting, StatusUploading}, // Added StatusUploading for auto-upload
	StatusError:      {StatusWaiting},                  // For retry
	StatusProcessing: {StatusActive, StatusError, StatusWaiting},
	"":               {StatusWaiting}, // Initial state
}

func CanTransition(from, to DownloadStatus) bool {
	allowed, exists := ValidTransitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func ValidateTransition(from, to DownloadStatus) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid transition from %s to %s", from, to)
	}
	return nil
}
