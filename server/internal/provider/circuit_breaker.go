package provider

import (
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	name         string
	state        CircuitState
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
	successCount int
	halfOpenMax  int
}

func NewCircuitBreaker(name string, threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		state:        StateClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
		halfOpenMax:  2,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successCount < cb.halfOpenMax
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.state = StateClosed
			cb.failures = 0
		}
	} else {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = StateOpen
	}
}
