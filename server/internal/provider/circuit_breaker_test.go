package provider

import (
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-provider", 5, 30*time.Second)

	// Test initial state
	if !cb.Allow() {
		t.Error("Circuit breaker should allow requests initially")
	}

	// Record failures below threshold
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
		if !cb.Allow() {
			t.Errorf("Circuit breaker should allow requests after %d failures", i+1)
		}
	}

	// Record 5th failure - should open circuit
	cb.RecordFailure()
	if cb.Allow() {
		t.Error("Circuit breaker should NOT allow requests after 5 failures")
	}

	// Record success while open (without timeout) - should stay open
	cb.RecordSuccess()
	// Need to check state or Allow() again. RecordSuccess resets failures if Closed/HalfOpen,
	// but if Open it does nothing unless we transitioned?
	// The implementation of RecordSuccess:
	// if HalfOpen -> count success -> Close
	// else -> failures = 0

	// WAIT: If state is OPEN, RecordSuccess sets failures = 0?
	// Let's check logic:
	/*
		if cb.state == StateHalfOpen {
			...
		} else {
			cb.failures = 0
		}
	*/
	// This means if we call RecordSuccess while OPEN, it resets failure count but DOES NOT change state back to Closed.
	// But Allow() checks state first.
	// So if state is OPEN, failures=0 doesn't matter, it still checks time.Since(lastFailure).

	if cb.Allow() {
		t.Error("Circuit breaker should stay open until timeout, even if Success recorded prematurely")
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := NewCircuitBreaker("test-provider", 5, 30*time.Second)

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.Allow() {
		t.Fatal("Should be open")
	}

	// Simulate timeout expiry by modifying private field
	cb.mu.Lock()
	cb.lastFailure = time.Now().Add(-31 * time.Second)
	cb.mu.Unlock()

	// Now should allow (and transition to HalfOpen)
	if !cb.Allow() {
		t.Error("Should allow after timeout")
	}

	cb.mu.RLock()
	if cb.state != StateHalfOpen {
		t.Errorf("State should be HalfOpen, got %v", cb.state)
	}
	cb.mu.RUnlock()

	// Record successes to close
	cb.RecordSuccess() // 1
	cb.RecordSuccess() // 2 (max is 2 hardcoded)

	cb.mu.RLock()
	if cb.state != StateClosed {
		t.Errorf("State should be Closed after successes, got %v", cb.state)
	}
	cb.mu.RUnlock()
}
