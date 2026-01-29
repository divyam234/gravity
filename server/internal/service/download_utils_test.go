package service

import (
	"errors"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"Timeout error", errors.New("connection timed out"), true},
		{"Connection refused", errors.New("dial tcp: connection refused"), true},
		{"Rate limit (429)", errors.New("server returned 429 Too Many Requests"), true},
		{"Service unavailable (503)", errors.New("503 Service Unavailable"), true},
		{"Bad gateway (502)", errors.New("502 Bad Gateway"), true},
		{"Generic error", errors.New("file not found"), false},
		{"Validation error", errors.New("invalid input"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		want       time.Duration
	}{
		{0, 30 * time.Second},
		{1, 60 * time.Second},
		{2, 120 * time.Second},
		{3, 240 * time.Second},
		{5, 960 * time.Second},   // 16 min
		{6, 1800 * time.Second},  // 32 min -> capped at 30 min
		{10, 1800 * time.Second}, // capped
	}

	for _, tt := range tests {
		t.Run("retry "+string(rune(tt.retryCount)), func(t *testing.T) {
			if got := calculateBackoff(tt.retryCount); got != tt.want {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.retryCount, got, tt.want)
			}
		})
	}
}
